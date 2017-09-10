package tcmu

import (
	"errors"
	"fmt"

	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
	"github.com/sirupsen/logrus"
)

type nlink struct {
	c      *genetlink.Conn
	family genetlink.Family
}

var (
	netlink_unsupported = errors.New("netlink is not supported")
)

// tcmu_genl_attr
// include/uapi/linux/target_core_user.h
const (
	TCMU_ATTR_UNSPEC = iota
	TCMU_ATTR_DEVICE
	TCMU_ATTR_MINOR
	TCMU_ATTR_PAD
	TCMU_ATTR_DEV_CFG
	TCMU_ATTR_DEV_SIZE
	TCMU_ATTR_WRITECACHE
	TCMU_ATTR_CMD_STATUS
	TCMU_ATTR_DEVICE_ID
	TCMU_ATTR_SUPP_KERN_CMD_REPLY
)

// tcmu_genl_cmd
// include/uapi/linux/target_core_user.h
const (
	TCMU_CMD_UNSPEC = iota
	TCMU_CMD_ADDED_DEVICE
	TCMU_CMD_REMOVED_DEVICE
	TCMU_CMD_RECONFIG_DEVICE
	TCMU_CMD_ADDED_DEVICE_DONE
	TCMU_CMD_REMOVED_DEVICE_DONE
	TCMU_CMD_RECONFIG_DEVICE_DONE
	TCMU_CMD_SET_FEATURES
)

// setNetlink creates netlink connection and enables netlink command reply.
func setNetlink() (*nlink, error) {
	c, err := genetlink.Dial(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to dial netlink: %v", err)
	}

	n := &nlink{c: c}
	n.family, err = c.GetFamily("TCM-USER")
	if err != nil {
		return n, fmt.Errorf("not found TCM-USER netink. you might miss to load target_core_user kernel module")
	}
	var groupID uint32
	for _, g := range n.family.Groups {
		if g.Name == "config" {
			groupID = n.family.Groups[0].ID
			break
		}
	}
	if groupID == 0 {
		return n, fmt.Errorf("not found groupdID")
	}

	// kernel supports tcmu netlink reply v2 or later.
	if n.family.Version < 2 {
		logrus.Info("netlink communication is disabled, as kernel does not support it")
		return n, nil
	}

	err = c.JoinGroup(groupID)
	if err != nil {
		return n, fmt.Errorf("failed to join group: %v", err)
	}

	a := []netlink.Attribute{{
		Type: TCMU_ATTR_SUPP_KERN_CMD_REPLY,
		Data: enabled(),
	}}
	attr, err := netlink.MarshalAttributes(a)
	if err != nil {
		return n, fmt.Errorf("Failed to marshal netlink attributes: %v", err)
	}

	req := genetlink.Message{
		Header: genetlink.Header{
			Command: TCMU_CMD_SET_FEATURES,
			Version: n.family.Version,
		},
		Data: attr,
	}
	_, err = c.Send(req, n.family.ID, netlink.HeaderFlagsRequest)
	if err != nil {
		return n, fmt.Errorf("failed to enabled netlink: %v", err)
	}
	return n, nil
}

// handleNetlink handles netlink command from kernel.
func (n *nlink) handleNetlink() error {
	for {
		msgs, _, err := n.c.Receive()
		if err != nil {
			logrus.Errorf("failed to receive netlink: %v\n", err)
			continue
		}

		if len(msgs) != 1 {
			logrus.Errorf("received unexpected messages: %#v\n", msgs)
			continue
		}

		atbs, err := netlink.UnmarshalAttributes(msgs[0].Data)
		if err != nil {
			logrus.Errorf("failed to unmarshal received message %#v: %v\n", msgs[0].Data, err)
			continue
		}

		deviceID := make([]byte, 4)
		for i, _ := range atbs {
			if atbs[i].Type == TCMU_ATTR_DEVICE_ID {
				deviceID = atbs[i].Data
			}
		}

		var replyCmd uint8
		var result int32
		switch msgs[0].Header.Command {
		case TCMU_CMD_ADDED_DEVICE:
			// TODO
			// somehting and status = 0
			result = 0
			replyCmd = TCMU_CMD_ADDED_DEVICE_DONE
			err = n.handleNetlinkReply(result, deviceID, replyCmd)
		case TCMU_CMD_REMOVED_DEVICE:
			// TODO
			// somehting and status = 0
			result = 0
			replyCmd = TCMU_CMD_REMOVED_DEVICE_DONE
			err = n.handleNetlinkReply(result, deviceID, replyCmd)
			return nil
		case TCMU_CMD_RECONFIG_DEVICE:
			//TODO
			// somehting and status = 0
			result = 0
			replyCmd = TCMU_CMD_RECONFIG_DEVICE_DONE
			err = n.handleNetlinkReply(result, deviceID, replyCmd)
		default:
			logrus.Errorf("received unexpected command %#v\n", msgs[0])
			continue
		}
		if err != nil {

		}
	}
}

// handleNetlinkReply replys netlink command.
func (n *nlink) handleNetlinkReply(s int32, deviceID []byte, done_cmd uint8) error {
	status := make([]byte, 4)
	nlenc.PutInt32(status, s)

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
			Data: deviceID,
		},
	}
	data, err := netlink.MarshalAttributes(attrs)
	if err != nil {
		logrus.Errorf("failed to marshal attributes %#v: %v\n", attrs, err)
	}

	req := genetlink.Message{
		Header: genetlink.Header{
			Command: done_cmd,
			Version: n.family.Version,
		},
		Data: data,
	}
	_, err = n.c.Send(req, n.family.ID, netlink.HeaderFlagsRequest)
	if err != nil {
		logrus.Fatalf("failed to send request: %v\n", err)
	}
	return err
}

// enabled creates a byte slice with 1.
func enabled() []byte {
	o := make([]byte, 1)
	nlenc.PutUint8(o, 1)
	return o
}
