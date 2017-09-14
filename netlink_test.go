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
	//	"github.com/mdlayher/netlink/nltest"
)

func TestHandleNetlink(t *testing.T) {
	const (
		length = 24
		flags  = netlink.HeaderFlagsRequest
	)
	/*
		groupID := uint32(1000)
		f := genetlink.Family{
			ID:      1000, // ID could be random
			Name:    "config",
			Version: 2,
			Groups:  []genetlink.MulticastGroup{{ID: groupID}},
		}
		c := genltest.Dial(func(creq genetlink.Message, _ netlink.Message) ([]genetlink.Message, error) {
			// Turn the request back around to the client.
			return []genetlink.Message{creq}, nil
		})
		defer c.Close()
		// c := genltest.Dial(func(_ genetlink.Message, _ netlink.Message) ([]genetlink.Message, error) {
		// 	return nil, io.EOF
		// })
		// defer c.Close()
	*/
	n, _ := NewNetlink()

	//	n := &nlink{c, f}

	// 1. create connection (Dial)
	// 2. run handleNetlink()
	// 3. send a test data

	// 1. create connection (Dial)

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
		//		logrus.Errorf("failed to marshal attributes %#v: %v\n", attrs, err)
	}
	req := genetlink.Message{
		Header: genetlink.Header{
			Command: TCMU_CMD_FOR_TEST,
			Version: 2,
		},
		Data: data,
	}

	/*
		want := netlink.Message{
			Header: netlink.Header{
				Length: length,
				//			Type:   f.ID,
				Flags: flags,
				PID:   nltest.PID,
			},
			Data: mustMarshal(req),
		}
	*/

	// 2. run handleNetlink()
	go func() {
		if err := n.handleNetlink(); err != nil {
			t.Fatalf("failed to receive: %v", err)
		} else {
			t.Fatalf("ok")
		}
	}()

	// 3. send a test data (emulate kernel's netlink message.)
	nlreq, err := n.c.Send(req, n.family.ID, flags)
	if err != nil {
		t.Fatalf("failed to send: %v, %v", err, nlreq)
	}
	// TODO wait
	time.Sleep(1000000)
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
