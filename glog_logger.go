/**
 * Copyright (c) 2016-present, Facebook, Inc.
 * All rights reserved.
 *
 * This source code is licensed under the BSD-style license found in the
 * LICENSE file in the root directory of this source tree. An additional grant
 * of patent rights can be found in the PATENTS file in the same directory.
 */

package main

import (
	"fmt"
	"github.com/facebookincubator/dhcplb/lib"
	"github.com/golang/glog"
	"github.com/krolaw/dhcp4"
)

type glogLogger struct{}

// NewGlogLogger returns a glogLogger struct based on the
// dhcplb.PersonalizedLogger interface.
func NewGlogLogger() dhcplb.PersonalizedLogger {
	return glogLogger{}
}

// Log takes a dhcplb.LogMessage, creates a sample map[string] containing
// information about the served request and prints it to stdout/err.
func (l glogLogger) Log(msg dhcplb.LogMessage) error {
	sample := map[string]interface{}{
		"version":      msg.Version,
		"dhcp_server":  msg.Server,
		"server_is_rc": msg.ServerIsRC,
		"source_ip":    msg.Peer.IP.String(),
		"success":      msg.Success,
		"latency_us":   msg.Latency.Nanoseconds() / 1000,
	}
	if msg.ErrorName != "" {
		sample["error_name"] = msg.ErrorName
		sample["error_details"] = fmt.Sprintf("%s", msg.ErrorDetails)
	}

	if msg.Packet != nil {
		if msg.Version == 4 {
			packet := dhcp4.Packet(msg.Packet)
			t := dhcp4.MessageType(
				packet.ParseOptions()[dhcp4.OptionDHCPMessageType][0])
			sample["type"] = t.String()
			sample["xid"] = fmt.Sprintf("%#06x", packet.XId())
			sample["giaddr"] = packet.GIAddr().String()
			sample["client_mac"] = packet.CHAddr().String()
		} else if msg.Version == 6 {
			packet := dhcplb.Packet6(msg.Packet)
			sample["type"] = packet.Type().String()
			xid, _ := packet.XID()
			sample["xid"] = fmt.Sprintf("%#06x", xid)
			duid, _ := packet.Duid()
			sample["duid"] = dhcplb.FormatID(duid)
			mac, err := packet.Mac()
			if err != nil {
				glog.Errorf("error getting mac: %s", err)
			}
			sample["client_mac"] = dhcplb.FormatID(mac)
			link, _ := packet.LinkAddr()
			sample["link-addr"] = link.String()
			peer, _ := packet.PeerAddr()
			sample["peer-addr"] = peer.String()
		}
	}

	if msg.Success {
		glog.Infof("%+v", sample)
	} else {
		glog.Errorf("%+v", sample)
	}
	return nil
}
