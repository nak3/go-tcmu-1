package tcmu

import (
	//	"errors"
	"encoding"
	"fmt"
	//	"io"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/genetlink"
	//"github.com/mdlayher/genetlink/genltest"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
	"github.com/mdlayher/netlink/nltest"
)

func TestHandleNetlink(t *testing.T) {
	const (
		length = 0x30
		flags  = netlink.HeaderFlagsRequest
	)
	n, _ := TempNewNetlink()

	received := false
	go func() {
		if err := n.handleNetlink(); err != nil {
			t.Fatalf("Failed to handleNetlink: %v", err)
		} else {
			received = true
			n.doneCh <- struct{}{}
		}
	}()

	minor := make([]byte, 4)
	nlenc.PutUint32(minor, 1)

	devID := make([]byte, 4)
	nlenc.PutUint32(devID, 1)

	attrs := []netlink.Attribute{
		{
			Type: TCMU_ATTR_DEVICE,
			Data: []byte(nlenc.Bytes("foo/bar")),
		},
		{
			Type: TCMU_ATTR_MINOR,
			Data: minor,
		},
		{
			Type: TCMU_ATTR_DEVICE_ID,
			Data: devID,
		},
	}

	data, err := netlink.MarshalAttributes(attrs)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}
	req := genetlink.Message{
		Header: genetlink.Header{
			Command: TCMU_CMD_FOR_TEST,
			Version: 2,
		},
		Data: data,
	}

	want := netlink.Message{
		Header: netlink.Header{
			Length: length,
			Flags:  flags,
			PID:    nltest.PID,
		},
		Data: mustMarshal(req),
	}

	go func() {
		select {
		case <-n.doneCh:
			return
		}
	}()

	nlreq, err := n.c.Send(req, n.family.ID, flags)
	if err != nil {
		t.Fatalf("failed to send: %v, %v", err, nlreq)
	}

	if diff := diffNetlinkMessages(want, nlreq); diff != "" {
		t.Fatalf("unexpected returned netlink message (-want +got):\n%s", diff)
	}

	time.Sleep(3000 * time.Millisecond)
	if !received {
		t.Fatalf("expected netlink received data, but not received")
	}
}

func TestHandleReplyNetlink(t *testing.T) {

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
