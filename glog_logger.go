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
	"sort"
	"strings"

	"github.com/facebookincubator/dhcplb/lib"
	"github.com/golang/glog"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv6"
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
			packet, _ := dhcpv4.FromBytes(msg.Packet)
			for _, opt := range packet.Options() {
				if opt.Code() == dhcpv4.OptionDHCPMessageType {
					sample["type"] = opt.String()
					sample["xid"] = fmt.Sprintf("%#06x", packet.TransactionID())
					sample["giaddr"] = packet.GatewayIPAddr().String()
					break
				}
			}
			sample["client_mac"] = packet.ClientHwAddrToString()
		} else if msg.Version == 6 {
			packet, err := dhcpv6.FromBytes(msg.Packet)
			if err != nil {
				glog.Errorf("Error encoding DHCPv6 packet: %s", err)
				return err
			}
			pt := dhcpv6.MessageTypeToString(packet.Type())
			sample["type"] = pt
			msg := packet
			if msg.IsRelay() {
				msg, err = msg.(*dhcpv6.DHCPv6Relay).GetInnerMessage()
				if err != nil {
					glog.Errorf("Failed to get inner packet: %s", err)
					return err
				}
			}
			xid := msg.(*dhcpv6.DHCPv6Message).TransactionID()
			sample["xid"] = fmt.Sprintf("%#06x", xid)
			optclientid := msg.GetOneOption(dhcpv6.OPTION_CLIENTID)
			if optclientid != nil {
				duid := optclientid.(*dhcpv6.OptClientId).Cid
				sample["duid"] = dhcplb.FormatID(duid.ToBytes())
				mac := duid.LinkLayerAddr
				if mac == nil {
					mac, err = dhcplb.Mac(packet)
					if err != nil {
						glog.Errorf("error getting mac: %s", err)
					}
				}
				sample["client_mac"] = dhcplb.FormatID(mac)
				duidtypename, ok := dhcpv6.DuidTypeToString[duid.Type]
				if ok {
					sample["duid_type"] = duidtypename
				}
			}
			if packet.IsRelay() {
				relay := packet.(*dhcpv6.DHCPv6Relay)
				sample["link-addr"] = relay.LinkAddr().String()
				sample["peer-addr"] = relay.PeerAddr().String()
			}
		}
	}

	// Order samples by key, store them into logline slice
	keys := make([]string, len(sample))
	i := 0
	for key, _ := range sample {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	logline := make([]string, len(sample))
	i = 0
	for k := range keys {
		logline[i] = fmt.Sprintf("%s: %+v", keys[k], sample[keys[k]])
		i++
	}

	if msg.Success {
		glog.Infof("%s", strings.Join(logline, ", "))
	} else {
		glog.Errorf("%s", strings.Join(logline, ", "))
	}
	return nil
}
