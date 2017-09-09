package tcmu

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
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

func handleNetlink() error {
	c, err := genetlink.Dial(nil)
	if err != nil {
		return fmt.Errorf("failed to dial netlink: %v", err)
	}
	defer c.Close()

	family, err := c.GetFamily("TCM-USER")
	if err != nil {
		return fmt.Errorf("not found TCM-USER netink. you might miss to load target_core_user kernel module")
	}
	var groupID uint32
	for _, g := range family.Groups {
		if g.Name == "config" {
			groupID = family.Groups[0].ID
			break
		}
	}
	if groupID == 0 {
		return fmt.Errorf("not found groupdID")
	}

	// TODO
	// kernel does supports tcmu netlink v2 or later.
	if family.Version < 2 {
		logrus.Info("netlink communication is disabled, as kernel does not support it")
		return nil
	}

	if err := c.JoinGroup(groupID); err != nil {
		logrus.Fatalf("failed to join group: %v", err)
	}

	a := []netlink.Attribute{{
		Type: TCMU_ATTR_SUPP_KERN_CMD_REPLY,
		Data: enabled(),
	}}
	attr, err := netlink.MarshalAttributes(a)
	if err != nil {
		return fmt.Errorf("TODO")
	}

	req := genetlink.Message{
		Header: genetlink.Header{
			Command: TCMU_CMD_SET_FEATURES,
			Version: family.Version,
		},
		Data: attr,
	}
	_, err = c.Send(req, family.ID, netlink.HeaderFlagsRequest)
	if err != nil {
		logrus.Fatalf("failed to enabled to netlink: %v", err)
		return err
	}

	for {
		msgs, _, err := c.Receive()
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
			logrus.Errorf("failed to unmarshal received message: %v\n", err)
			continue
		}

		deviceID := make([]byte, 4)
		for i, _ := range atbs {
			if atbs[i].Type == TCMU_ATTR_DEVICE_ID {
				deviceID = atbs[i].Data
			}
		}

		var replyCmd uint8
		var result uint32
		switch msgs[0].Header.Command {
		case TCMU_CMD_ADDED_DEVICE:
			//TODO
			// somehting and status = 0
			result = 0
			replyCmd = TCMU_CMD_ADDED_DEVICE_DONE
		case TCMU_CMD_REMOVED_DEVICE:
			//TODO
			// somehting and status = 0
			result = 0
			replyCmd = TCMU_CMD_REMOVED_DEVICE_DONE
			return nil
		case TCMU_CMD_RECONFIG_DEVICE:
			//TODO
			// somehting and status = 0
			result = 0
			replyCmd = TCMU_CMD_RECONFIG_DEVICE_DONE
		default:
			logrus.Errorf("received unexpected command %#v", msgs[0])
			continue
		}
		handleNetlinkReply(c, &family, result, deviceID, replyCmd)
	}
}

func handleNetlinkReply(c *genetlink.Conn, family *genetlink.Family, s uint32, deviceID []byte, done_cmd uint8) error {
	status := make([]byte, 4)
	nlenc.PutUint32(status, s)

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
		logrus.Errorf("failed to marshal attributes: %v", err)
	}

	req := genetlink.Message{
		Header: genetlink.Header{
			Command: done_cmd,
			Version: family.Version,
		},
		Data: data,
	}
	_, err = c.Send(req, family.ID, netlink.HeaderFlagsRequest)
	if err != nil {
		logrus.Fatalf("failed to send request: %v", err)
	}
	return err
}

func enabled() []byte {
	o := make([]byte, 1)
	nlenc.PutUint8(o, 1)
	return o
}
