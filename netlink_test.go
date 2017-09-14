package tcmu

import (
	"encoding"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
	"github.com/mdlayher/netlink/nltest"
)

func TestSetNetlink(t *testing.T) {
	t.Parallel()
	n, _ := TempNewNetlink()

	data, err := netlink.MarshalAttributes([]netlink.Attribute{
		{
			Type: TCMU_ATTR_SUPP_KERN_CMD_REPLY,
			Data: enabled(),
		}},
	)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}
	want := []genetlink.Message{{
		Header: genetlink.Header{
			Command: TCMU_CMD_SET_FEATURES,
			Version: n.family.Version,
		},
		Data: data,
	}}
	received := false
	go func() {
		if msgs, _, err := n.c.Receive(); err != nil {
			t.Fatalf("Failed to Receive: %v", err)
		} else {
			received = true
			if diff := cmp.Diff(want, msgs); diff != "" {
				t.Fatalf("unexpected replies (-want +got):\n%s", diff)
			}
		}
	}()

	if err := n.setNetlink(); err != nil {
		t.Fatalf("Failed set netlink feature: %v", err)
	}

	// Wait 1 sec for receiving netlink.
	time.Sleep(1000 * time.Millisecond)

	if !received {
		t.Fatalf("expected netlink received data, but not received")
	}

}

func TestHandleNetlink(t *testing.T) {
	t.Parallel()
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
			Version: n.family.Version,
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

	nlreq, err := n.c.Send(req, n.family.ID, flags)
	if err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	if diff := diffNetlinkMessages(want, nlreq); diff != "" {
		t.Fatalf("unexpected returned netlink message (-want +got):\n%s", diff)
	}

	// Wait 1 sec for receiving netlink.
	time.Sleep(1000 * time.Millisecond)

	if !received {
		t.Fatalf("expected netlink received data, but not received")
	}
}

func TestHandleReplyNetlink(t *testing.T) {
	t.Parallel()
	n, _ := TempNewNetlink()

	var ok int32 = 0

	devID := make([]byte, 4)
	nlenc.PutUint32(devID, 1)

	status := make([]byte, 4)
	nlenc.PutInt32(status, ok)

	attrs := []netlink.Attribute{
		{
			Type: TCMU_ATTR_SUPP_KERN_CMD_REPLY,
			Data: enabled(),
		},
		{
			Type: TCMU_ATTR_CMD_STATUS,
			Data: status,
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

	want := []genetlink.Message{{
		Header: genetlink.Header{
			Command: TCMU_CMD_FOR_TEST_DONE,
			Version: n.family.Version,
		},
		Data: data,
	}}

	received := false
	go func() {
		if msgs, _, err := n.c.Receive(); err != nil {
			t.Fatalf("Failed to Receive: %v", err)
		} else {
			received = true
			if diff := cmp.Diff(want, msgs); diff != "" {
				t.Fatalf("unexpected replies (-want +got):\n%s", diff)
			}
		}
	}()

	err = n.handleNetlinkReply(ok, devID, TCMU_CMD_FOR_TEST_DONE)
	if err != nil {
		t.Fatalf("Failed to handleNetlinkReply: %v", err)
	}

	// Wait 1 sec for receiving netlink.
	time.Sleep(1000 * time.Millisecond)

	if !received {
		t.Fatalf("expected netlink received data, but not received")
	}
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
