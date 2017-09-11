package tcmu

import (
	//	"errors"
	"encoding"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/genetlink/genltest"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nltest"
)

func TestHandleNetlink(t *testing.T) {
	const (
		length = 24
		flags  = netlink.HeaderFlagsRequest
	)
	f := genetlink.Family{
		ID:      1,
		Name:    "config",
		Version: 2,
	}
	// 1. create connection (Dial)
	// 2. run handleNetlink()
	// 3. send a test data

	// 1. create connection (Dial)

	attrs := []netlink.Attribute{
		{
			Type: TCMU_ATTR_DEVICE_ID,
			Data: []byte("foo/bar"),
		},
	}
	data, err := netlink.MarshalAttributes(attrs)
	if err != nil {
		//		logrus.Errorf("failed to marshal attributes %#v: %v\n", attrs, err)
	}
	req := genetlink.Message{
		Header: genetlink.Header{
			Command: TCMU_CMD_ADDED_DEVICE,
			//	Command: TCMU_CMD_REMOVED_DEVICE,
			Version: 2,
		},
		// TODO: Device ID
		Data: data,
	}

	want := netlink.Message{
		Header: netlink.Header{
			Length: length,
			//			Type:   f.ID,
			Flags: flags,
			PID:   nltest.PID,
		},
		Data: mustMarshal(req),
	}

	c := genltest.Dial(func(_ genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		if diff := diffNetlinkMessages(want, nreq); diff != "" {
			//			t.Fatalf("unexpected sent netlink message (-want +got):\n%s", diff)
		}
		return nil, nil
	})
	c, _ = genetlink.Dial(nil)

	n := &nlink{c, f}

	// 2. run handleNetlink()
	go func() {
		if err := n.handleNetlink(); err != nil {
			t.Fatalf("failed to receive: %v", err)
		} else {
			t.Fatalf("done receive: %v", err)
		}
	}()

	// 3. send a test data (emulate kernel's netlink message.)
	nlreq, err := n.c.Send(req, f.ID, flags)
	if err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	fmt.Println(nlreq)

	if diff := diffNetlinkMessages(want, nlreq); diff != "" {
		//		t.Fatalf("unexpected returned netlink message (-want +got):\n%s", diff)
	}
	time.Sleep(1)
}

func TestHandleNetlink(t *testing.T) {
}
func mustMarshal(m encoding.BinaryMarshaler) []byte {
	b, err := m.MarshalBinary()
	if err != nil {
		panic(fmt.Sprintf("failed to marshal binary: %v", err))
	}

	return b
}

// diffNetlinkMessages compares two netlink.Messages after zeroing their
// sequence number fields that make equality checks in testing difficult.
func diffNetlinkMessages(want, got netlink.Message) string {
	want.Header.Sequence = 0
	got.Header.Sequence = 0

	return cmp.Diff(want, got)
}
