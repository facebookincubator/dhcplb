/**
 * Copyright (c) 2016-present, Facebook, Inc.
 * All rights reserved.
 *
 * This source code is licensed under the BSD-style license found in the
 * LICENSE file in the root directory of this source tree. An additional grant
 * of patent rights can be found in the PATENTS file in the same directory.
 */

package dhcplb

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/mdlayher/eui64"
)

// List of possible errors.
const (
	ErrUnknown  = "E_UNKNOWN"
	ErrPanic    = "E_PANIC"
	ErrRead     = "E_READ"
	ErrConnect  = "E_CONN"
	ErrWrite    = "E_WRITE"
	ErrGi0      = "E_GI_0"
	ErrParse    = "E_PARSE"
	ErrNoServer = "E_NO_SERVER"
	ErrConnRate = "E_CONN_RATE"
)

func handleConnection(conn *net.UDPConn, config *Config, logger loggerHelper, bufPool *sync.Pool, throttle Throttle) {
	buffer := bufPool.Get().([]byte)
	bytesRead, peer, err := conn.ReadFromUDP(buffer)
	if err != nil || bytesRead == 0 {
		bufPool.Put(buffer)
		msg := "error reading from %s: %v"
		glog.Errorf(msg, peer, err)
		logger.LogErr(time.Now(), nil, nil, peer, ErrRead, err)
		return
	}

	go func() {
		defer func() {
			// always release this routine's buffer back to the pool
			bufPool.Put(buffer)

			if r := recover(); r != nil {
				glog.Errorf("Panicked handling v%d packet from %s: %s", config.Version, peer, r)
				glog.Errorf("Offending packet: %x", buffer[:bytesRead])
				err, _ := r.(error)
				logger.LogErr(time.Now(), nil, nil, peer, ErrPanic, err)
				glog.Errorf("%s: %s", r, debug.Stack())
			}
		}()

		if config.Version == 4 {
			handleRawPacketV4(logger, config, buffer[:bytesRead], peer, throttle)
		} else if config.Version == 6 {
			handleRawPacketV6(logger, config, buffer[:bytesRead], peer, throttle)
		}
	}()
}

// FormatID takes a list of bytes and formats them for printing.
// E.g. []byte{0x12, 0x34, 0x56, 0x78, 0x9a} will be printed as "12:34:56:78:9a"
func FormatID(buf []byte) string {
	if buf == nil || len(buf) == 0 {
		return ""
	}
	str := make([]byte, len(buf)*3-1)
	for i := 0; i < len(buf); i++ {
		strIndex := i * 3
		hex.Encode(str[strIndex:strIndex+2], []byte{buf[i]})
		if i < len(buf)-1 {
			str[strIndex+2] = ':'
		}
	}
	return string(str)
}

// Mac looks into the inner most PeerAddr field in the RelayInfo header.
// This contains the EUI-64 address of the client making the request, populated
// by the dhcp relay, it is possible to extract the mac address from that IP.
// If a mac address cannot be found an error will be returned.
func Mac(packet dhcpv6.DHCPv6) ([]byte, error) {
	if !packet.IsRelay() {
		return nil, fmt.Errorf("It is not possible to get the inner most relay")
	}
	ip, err := packet.(*dhcpv6.DHCPv6Relay).GetInnerPeerAddr()
	if err != nil {
		return nil, err
	}
	_, mac, err := eui64.ParseIP(ip)
	if err != nil {
		return nil, err
	}
	return mac, nil
}

func selectDestinationServer(config *Config, message *DHCPMessage) (*DHCPServer, error) {
	server, err := handleOverride(config, message)
	if err != nil {
		glog.Errorf("Error handling override, drop due to: %s", err)
		return nil, err
	}
	if server == nil {
		server, err = config.Algorithm.SelectRatioBasedDhcpServer(message)
	}
	return server, err
}

func handleOverride(config *Config, message *DHCPMessage) (*DHCPServer, error) {
	if override, ok := config.Overrides[FormatID(message.Mac)]; ok {
		// Checking if override is expired. If so, ignore it. Expiration field should
		// be a timestamp in the following format "2006/01/02 15:04 -0700".
		// For example, a timestamp in UTC would look as follows: "2017/05/06 14:00 +0000".
		var err error
		var expiration time.Time
		if override.Expiration != "" {
			expiration, err = time.Parse("2006/01/02 15:04 -0700", override.Expiration)
			if err != nil {
				glog.Errorf("Could not parse override expiration for MAC %s: %s", FormatID(message.Mac), err.Error())
				return nil, nil
			}
			if time.Now().After(expiration) {
				glog.Errorf("Ovverride rule for MAC %s expired on %s, ignoring", FormatID(message.Mac), expiration.Local())
				return nil, nil
			}
		}
		if override.Expiration == "" {
			glog.Infof("Found override rule for %s without expiration", FormatID(message.Mac))
		} else {
			glog.Infof("Found override rule for %s, it will expire on %s", FormatID(message.Mac), expiration.Local())
		}

		var server *DHCPServer
		if len(override.Host) > 0 {
			server, err = handleHostOverride(config, override.Host)
		} else if len(override.Tier) > 0 {
			server, err = handleTierOverride(config, override.Tier, message)
		}
		if err != nil {
			return nil, err
		}
		if server != nil {
			server.connect()
			time.AfterFunc(config.FreeConnTimeout, func() {
				err := server.disconnect()
				if err != nil {
					glog.Errorf("Failed to disconnect from %s", server)
				}
			})
			return server, nil
		}
		glog.Infof("Override didn't have host or tier, this shouldn't happen, proceeding with normal server selection")
	}
	return nil, nil
}

func handleHostOverride(config *Config, host string) (*DHCPServer, error) {
	addr := net.ParseIP(host)
	if addr == nil {
		return nil, fmt.Errorf("Failed to get IP for overridden host %s", host)
	}
	port := 67
	if config.Version == 6 {
		port = 547
	}
	server := NewDHCPServer(host, addr, port)
	return server, nil
}

func handleTierOverride(config *Config, tier string, message *DHCPMessage) (*DHCPServer, error) {
	servers, err := config.HostSourcer.GetServersFromTier(tier)
	if err != nil {
		return nil, fmt.Errorf("Failed to get servers from tier: %s", err)
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("Sourcer returned no servers")
	}
	// pick server according to the configured algorithm
	server, err := config.Algorithm.SelectServerFromList(servers, message)
	if err != nil {
		return nil, fmt.Errorf("Failed to select server: %s", err)
	}
	return server, nil
}

func sendToServer(logger loggerHelper, start time.Time, server *DHCPServer, packet []byte, peer *net.UDPAddr, throttle Throttle) error {

	// Check for connection rate
	ok, err := throttle.OK(server.Address.String())
	if !ok {
		glog.Errorf("Error writing to server %s, drop due to throttling", server.Hostname)
		logger.LogErr(time.Now(), server, packet, peer, ErrConnRate, err)
		return err
	}

	err = server.sendTo(packet)
	if err != nil {
		glog.Errorf("Error writing to server %s, drop due to %s", server.Hostname, err)
		logger.LogErr(start, server, packet, peer, ErrWrite, err)
		return err
	}

	err = logger.LogSuccess(start, server, packet, peer)
	if err != nil {
		glog.Errorf("Failed to log request: %s", err)
	}

	return nil
}

func handleRawPacketV4(logger loggerHelper, config *Config, buffer []byte, peer *net.UDPAddr, throttle Throttle) {
	// runs in a separate go routine
	start := time.Now()
	var message DHCPMessage
	packet, err := dhcpv4.FromBytes(buffer)
	if err != nil {
		glog.Errorf("Error encoding DHCPv4 packet: %s", err)
		logger.LogErr(start, nil, packet.ToBytes(), peer, ErrParse, err)
		return
	}

	message.XID = packet.TransactionID()
	message.Peer = peer
	clientHwAddr := packet.ClientHwAddr()
	hwAddrLen := packet.HwAddrLen()
	message.ClientID = clientHwAddr[:hwAddrLen]
	message.Mac = clientHwAddr[:hwAddrLen]

	for _, o := range packet.Options() {
		if o.Code() == dhcpv4.OptionVendorSpecificInformation ||
			o.Code() == dhcpv4.OptionTFTPServerName ||
			o.Code() == dhcpv4.OptionBootfileName {
			message.NetBoot = true
			break
		}
	}

	packet.SetHopCount(packet.HopCount() + 1)

	message.VendorData = VendorDataV4(packet)

	server, err := selectDestinationServer(config, &message)
	if err != nil {
		glog.Errorf("%s, Drop due to %s", packet.Summary(), err)
		logger.LogErr(start, nil, packet.ToBytes(), peer, ErrNoServer, err)
		return
	}

	sendToServer(logger, start, server, packet.ToBytes(), peer, throttle)
}

func handleRawPacketV6(logger loggerHelper, config *Config, buffer []byte, peer *net.UDPAddr, throttle Throttle) {
	// runs in a separate go routine
	start := time.Now()
	packet, err := dhcpv6.FromBytes(buffer)
	if err != nil {
		glog.Errorf("Error encoding DHCPv6 packet: %s", err)
		logger.LogErr(start, nil, packet.ToBytes(), peer, ErrParse, err)
		return
	}

	if packet.Type() == dhcpv6.RELAY_REPL {
		handleV6RelayRepl(logger, start, packet, peer)
		return
	}

	var message DHCPMessage

	msg := packet
	if msg.IsRelay() {
		msg, err = msg.(*dhcpv6.DHCPv6Relay).GetInnerMessage()
		if err != nil {
			glog.Errorf("Error getting inner message: %s", err)
			logger.LogErr(start, nil, packet.ToBytes(), peer, ErrParse, err)
			return
		}
	}
	message.XID = msg.(*dhcpv6.DHCPv6Message).TransactionID()
	message.Peer = peer

	optclientid := msg.GetOneOption(dhcpv6.OPTION_CLIENTID)
	if optclientid == nil {
		glog.Errorf("Failed to extract Client ID, drop due to %s", err)
		logger.LogErr(start, nil, packet.ToBytes(), peer, ErrParse, err)
		return
	}
	duid := optclientid.(*dhcpv6.OptClientId).Cid
	message.ClientID = duid.ToBytes()
	mac := duid.LinkLayerAddr
	if mac == nil {
		mac, err = Mac(packet)
		if err != nil {
			glog.Errorf("Failed to extract MAC, drop due to %s", err)
			logger.LogErr(start, nil, packet.ToBytes(), peer, ErrParse, err)
			return
		}
	}
	message.Mac = mac

	optoro := msg.GetOneOption(dhcpv6.OPTION_ORO)
	if optoro != nil {
		for _, o := range optoro.(*dhcpv6.OptRequestedOption).RequestedOptions() {
			if o == dhcpv6.OPT_BOOTFILE_URL {
				message.NetBoot = true
				break
			}
		}
	}

	server, err := selectDestinationServer(config, &message)
	if err != nil {
		glog.Errorf("%s, Drop due to %s", packet.Summary(), err)
		logger.LogErr(start, nil, packet.ToBytes(), peer, ErrNoServer, err)
		return
	}

	relayMsg, err := dhcpv6.EncapsulateRelay(packet, dhcpv6.RELAY_FORW, net.IPv6zero, peer.IP)
	sendToServer(logger, start, server, relayMsg.ToBytes(), peer, throttle)
}

func handleV6RelayRepl(logger loggerHelper, start time.Time, packet dhcpv6.DHCPv6, peer *net.UDPAddr) {
	// when we get a relay-reply, we need to unwind the message, removing the top
	// relay-reply info and passing on the inner part of the message
	msg, err := dhcpv6.DecapsulateRelay(packet)
	if err != nil {
		glog.Errorf("Failed to decapsulate packet, drop due to %s", err)
		logger.LogErr(start, nil, packet.ToBytes(), peer, ErrParse, err)
		return
	}
	peerAddr := packet.(*dhcpv6.DHCPv6Relay).PeerAddr()
	// send the packet to the peer addr
	addr := &net.UDPAddr{
		IP:   peerAddr,
		Port: int(547),
		Zone: "",
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		glog.Errorf("Error creating udp connection %s", err)
		logger.LogErr(start, nil, packet.ToBytes(), peer, ErrConnect, err)
		return
	}
	conn.Write(msg.ToBytes())
	err = logger.LogSuccess(start, nil, packet.ToBytes(), peer)
	if err != nil {
		glog.Errorf("Failed to log request: %s", err)
	}
	conn.Close()
	return
}

var errVendorOptionMalformed = errors.New("malformed vendor option")

// VendorDataV4 will try to parse dhcp4 options data looking for more specific
// vendor data (like model, serial number, etc).  If the options are missing
func VendorDataV4(packet *dhcpv4.DHCPv4) VendorData {
	vd := VendorData{}

	for _, opt := range packet.Options() {
		switch opt.Code() {
		case dhcpv4.OptionClassIdentifier:
			if err := parseV4VendorClass(&vd, opt.(*dhcpv4.OptClassIdentifier)); err != nil {
				glog.Errorf("failed to parse vendor data from vendor class: %v", err)
			}
		case dhcpv4.OptionVendorIdentifyingVendorClass:
			if err := parseV4VIVC(&vd, opt.(*dhcpv4.OptVIVC)); err != nil {
				glog.Errorf("failed to parse vendor data from vendor-idenitifying vendor class: %v", err)
			}
		}

	}
	return vd
}

// parseV4Opt60 will attempt to look at the Vendor Class option (Option 60) on
// DHCPv4.  The option is formatted as a string with the content being specific
// for the vendor, usually using a deliminator to separate the values.
// See: https://tools.ietf.org/html/rfc1533#section-9.11
func parseV4VendorClass(vd *VendorData, opt *dhcpv4.OptClassIdentifier) error {
	vc := opt.Identifier
	switch {
	// Arista;DCS-7050S-64;01.23;JPE12221671
	case strings.HasPrefix(vc, "Arista;"):
		p := strings.Split(vc, ";")
		if len(p) < 4 {
			return errVendorOptionMalformed
		}

		vd.VendorName = p[0]
		vd.Model = p[1]
		vd.Serial = p[3]
		return nil

	// ZPESystems:NSC:002251623
	case strings.HasPrefix(vc, "ZPESystems:"):
		p := strings.Split(vc, ":")
		if len(p) < 3 {
			return errVendorOptionMalformed
		}

		vd.VendorName = p[0]
		vd.Model = p[1]
		vd.Serial = p[2]
		return nil

	// Juniper-ptx1000-DD576
	// Juniper also has cases where the model number may have a '-' in it as
	// well e.g.: Juniper-qfx10002-36q-DN817. Brillant Juniper. Brillant.
	case strings.HasPrefix(vc, "Juniper-"):
		idx := strings.Index(vc, "-")
		if idx == -1 {
			return errVendorOptionMalformed
		}
		lastIdx := strings.LastIndex(vc, "-")
		if lastIdx == -1 {
			return errVendorOptionMalformed
		}

		vd.VendorName = vc[0:idx]
		vd.Model = vc[idx+1 : lastIdx]
		vd.Serial = vc[lastIdx+1:]
		return nil
	}

	// We didn't match anything, just return an empty vendor data.
	return nil
}

const entIDCiscoSystems = 0x9

// parseV4Opt124 will attempt to read the Vendor-Identifying Vendor Class
// (Option 124) on a DHCPv4 packet.  The data is represented in a length/value
// format with an indentifying enterprise number.
//
// See: https://tools.ietf.org/html/rfc3925
func parseV4VIVC(vd *VendorData, opt *dhcpv4.OptVIVC) error {
	for _, id := range opt.Identifiers {
		if id.EntID == entIDCiscoSystems {
			vd.VendorName = "Cisco Systems"

			//SN:0;PID:R-IOSXRV9000-CC
			for _, f := range bytes.Split(id.Data, []byte(";")) {
				p := bytes.SplitN(f, []byte(":"), 2)
				if len(p) != 2 {
					return errVendorOptionMalformed
				}

				switch string(p[0]) {
				case "SN":
					vd.Serial = string(p[1])
				case "PID":
					vd.Model = string(p[1])
				}
			}
		}
	}
	return nil
}
