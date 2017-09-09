// tcmu is a package that connects to the TCM in Userspace kernel module, a part of the LIO stack. It provides the
// ability to emulate a SCSI storage device in pure Go.
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

func handleNetlink() {
	c, err := genetlink.Dial(nil)
	if err != nil {
		logrus.Fatalf("failed to dial netlink: %v", err)
		return
	}
	defer c.Close()
	family, err := c.GetFamily("TCM-USER")
	if err != nil {
		//TODO
		fmt.Printf("failed to get family:%v \n", err)
		return
	}
	var groupID uint32
	for _, g := range family.Groups {
		if g.Name == "config" {
			groupID = family.Groups[0].ID
			break
		}
	}
	if groupID == 0 {
		//TODO This must be not necessary as GetFamily already worked.
		logrus.Fatalf("not found")
	}

	one := make([]byte, 1)
	nlenc.PutUint8(one, 0)
	if err := c.JoinGroup(groupID); err != nil {
		logrus.Fatalf("failed to join group: %v", err)
	}

	a := []netlink.Attribute{{
		Type: TCMU_ATTR_SUPP_KERN_CMD_REPLY,
		Data: one,
	}}
	b, _ := netlink.MarshalAttributes(a)

	req := genetlink.Message{
		Header: genetlink.Header{
			Command: TCMU_CMD_SET_FEATURES,
			Version: family.Version,
		},
		Data: b,
	}
	_, err = c.Send(req, family.ID, netlink.HeaderFlagsRequest)
	if err != nil {
		logrus.Fatalf("failed to execute: %v", err)
	}

	for {
		logrus.Debugf("receiving: ...")
		msgs, _, err := c.Receive()
		if err != nil {
			fmt.Printf("failed to receive: %v \n", err)
			return
		}
		fmt.Printf(" %#v \n", msgs)
		atbs, _ := netlink.UnmarshalAttributes(msgs[0].Data)
		fmt.Printf(" %#v \n", atbs)
		deviceID := make([]byte, 4)

		for i, _ := range atbs {
			if atbs[i].Type == 0x8 {
				deviceID = atbs[i].Data
			}
			//fmt.Printf("---data -- > %s \n", atbs[i].Data)
		}
		fmt.Printf("---data -- > %#v \n", atbs[1].Data)

		switch msgs[0].Header.Command {
		case TCMU_CMD_ADDED_DEVICE:
			//TODO
			// somehting and status = 0
			handleNetlinkReply(c, &family, 0, deviceID, TCMU_CMD_ADDED_DEVICE_DONE)
		case TCMU_CMD_REMOVED_DEVICE:
			//TODO
			// somehting and status = 0
			handleNetlinkReply(c, &family, 0, deviceID, TCMU_CMD_REMOVED_DEVICE_DONE)
		case TCMU_CMD_RECONFIG_DEVICE:
			//TODO
			// somehting and status = 0
			handleNetlinkReply(c, &family, 0, deviceID, TCMU_CMD_RECONFIG_DEVICE_DONE)
		default:
			// error
		}

	}
}

func handleNetlinkReply(c *genetlink.Conn, family *genetlink.Family, s uint32, deviceID []byte, done_cmd uint8) error {
	status := make([]byte, 4)
	nlenc.PutUint32(status, s)
	one := make([]byte, 1)
	nlenc.PutUint8(one, 0)

	attrs := []netlink.Attribute{
		{
			Type: TCMU_ATTR_SUPP_KERN_CMD_REPLY,
			Data: one,
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
	data, _ := netlink.MarshalAttributes(attrs)

	req := genetlink.Message{
		Header: genetlink.Header{
			Command: done_cmd,
			Version: family.Version,
		},
		Data: data,
	}
	_, err := c.Send(req, family.ID, netlink.HeaderFlagsRequest)
	if err != nil {
		logrus.Fatalf("failed to execute: %v", err)
	}
	return err
}
