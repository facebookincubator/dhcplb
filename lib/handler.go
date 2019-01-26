/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package dhcplb

import (
	"errors"
	"fmt"
	"net"
	"runtime/debug"
	"time"

	"github.com/golang/glog"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/ztpv4"
	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/ztpv6"
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

func (s *serverImpl) handleConnection() {
	buffer := s.bufPool.Get().([]byte)
	bytesRead, peer, err := s.conn.ReadFromUDP(buffer)
	if err != nil || bytesRead == 0 {
		s.bufPool.Put(buffer)
		msg := "error reading from %s: %v"
		glog.Errorf(msg, peer, err)
		s.logger.LogErr(time.Now(), nil, nil, peer, ErrRead, err)
		return
	}

	go func() {
		defer func() {
			// always release this routine's buffer back to the pool
			s.bufPool.Put(buffer)

			if r := recover(); r != nil {
				glog.Errorf("Panicked handling v%d packet from %s: %s", s.config.Version, peer, r)
				glog.Errorf("Offending packet: %x", buffer[:bytesRead])
				err, _ := r.(error)
				s.logger.LogErr(time.Now(), nil, nil, peer, ErrPanic, err)
				glog.Errorf("%s: %s", r, debug.Stack())
			}
		}()

		if s.config.Version == 4 {
			s.handleRawPacketV4(buffer[:bytesRead], peer)
		} else if s.config.Version == 6 {
			s.handleRawPacketV6(buffer[:bytesRead], peer)
		}
	}()
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
	if override, ok := config.Overrides[message.Mac.String()]; ok {
		// Checking if override is expired. If so, ignore it. Expiration field should
		// be a timestamp in the following format "2006/01/02 15:04 -0700".
		// For example, a timestamp in UTC would look as follows: "2017/05/06 14:00 +0000".
		var err error
		var expiration time.Time
		if override.Expiration != "" {
			expiration, err = time.Parse("2006/01/02 15:04 -0700", override.Expiration)
			if err != nil {
				glog.Errorf("Could not parse override expiration for MAC %s: %s", message.Mac.String(), err.Error())
				return nil, nil
			}
			if time.Now().After(expiration) {
				glog.Errorf("Override rule for MAC %s expired on %s, ignoring", message.Mac.String(), expiration.Local())
				return nil, nil
			}
		}
		if override.Expiration == "" {
			glog.Infof("Found override rule for %s without expiration", message.Mac.String())
		} else {
			glog.Infof("Found override rule for %s, it will expire on %s", message.Mac.String(), expiration.Local())
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

func (s *serverImpl) handleRawPacketV4(buffer []byte, peer *net.UDPAddr) {
	// runs in a separate go routine
	start := time.Now()
	var message DHCPMessage
	packet, err := dhcpv4.FromBytes(buffer)
	if err != nil {
		glog.Errorf("Error encoding DHCPv4 packet: %s", err)
		s.logger.LogErr(start, nil, nil, peer, ErrParse, err)
		return
	}

	if s.server {
		s.handleV4Server(start, packet, peer)
		return
	}

	message.XID = packet.TransactionID
	message.Peer = peer
	message.ClientID = packet.ClientHWAddr
	message.Mac = packet.ClientHWAddr
	if vd, err := ztpv4.ParseVendorData(packet); err != nil {
		glog.V(2).Infof("error parsing vendor data: %s", err)
	} else {
		message.Serial = vd.Serial
	}

	if packet.Options.Has(dhcpv4.OptionVendorSpecificInformation) ||
		packet.Options.Has(dhcpv4.OptionTFTPServerName) ||
		packet.Options.Has(dhcpv4.OptionBootfileName) {
		message.NetBoot = true
	}

	packet.HopCount++

	server, err := selectDestinationServer(s.config, &message)
	if err != nil {
		glog.Errorf("%s, Drop due to %s", packet.Summary(), err)
		s.logger.LogErr(start, nil, packet.ToBytes(), peer, ErrNoServer, err)
		return
	}

	sendToServer(s.logger, start, server, packet.ToBytes(), peer, s.throttle)
}

func (s *serverImpl) handleV4Server(start time.Time, packet *dhcpv4.DHCPv4, peer *net.UDPAddr) {
	reply, err := s.config.Handler.ServeDHCPv4(packet)
	logErr := s.logger.LogSuccess(start, nil, packet.ToBytes(), peer)
	if logErr != nil {
		glog.Errorf("Failed to log incoming packet: %s", logErr)
	}
	if err != nil {
		glog.Errorf("Error creating reply %s", err)
		s.logger.LogErr(start, nil, packet.ToBytes(), peer, fmt.Sprintf("%T", err), err)
		return
	}
	addr := &net.UDPAddr{
		IP:   packet.GatewayIPAddr,
		Port: dhcpv4.ServerPort,
	}
	s.conn.WriteTo(reply.ToBytes(), addr)
	err = s.logger.LogSuccess(start, nil, reply.ToBytes(), peer)
	if err != nil {
		glog.Errorf("Failed to log reply: %s", err)
	}
	return
}

// temporary type until v6 changes can be merged.
type transactionID uint32

func (t transactionID) String() string {
	return fmt.Sprintf("%d", t)
}

func (s *serverImpl) handleRawPacketV6(buffer []byte, peer *net.UDPAddr) {
	// runs in a separate go routine
	start := time.Now()
	packet, err := dhcpv6.FromBytes(buffer)
	if err != nil {
		glog.Errorf("Error encoding DHCPv6 packet: %s", err)
		s.logger.LogErr(start, nil, nil, peer, ErrParse, err)
		return
	}

	if s.server {
		s.handleV6Server(start, packet, peer)
		return
	}

	if packet.Type() == dhcpv6.MessageTypeRelayReply {
		s.handleV6RelayRepl(start, packet, peer)
		return
	}

	var message DHCPMessage

	msg := packet
	if msg.IsRelay() {
		msg, err = msg.(*dhcpv6.DHCPv6Relay).GetInnerMessage()
		if err != nil {
			glog.Errorf("Error getting inner message: %s", err)
			s.logger.LogErr(start, nil, packet.ToBytes(), peer, ErrParse, err)
			return
		}
	}
	message.XID = transactionID(msg.(*dhcpv6.DHCPv6Message).TransactionID())
	message.Peer = peer

	optclientid := msg.GetOneOption(dhcpv6.OptionClientID)
	if optclientid == nil {
		errMsg := errors.New("Failed to extract Client ID")
		glog.Errorf("%v", errMsg)
		s.logger.LogErr(start, nil, packet.ToBytes(), peer, ErrParse, errMsg)
		return
	}
	duid := optclientid.(*dhcpv6.OptClientId).Cid
	message.ClientID = duid.ToBytes()
	mac, err := dhcpv6.ExtractMAC(packet)
	if err != nil {
		glog.Errorf("Failed to extract MAC, drop due to %s", err)
		s.logger.LogErr(start, nil, packet.ToBytes(), peer, ErrParse, err)
		return
	}
	message.Mac = mac
	message.NetBoot = msg.(*dhcpv6.DHCPv6Message).IsNetboot()
	if vendorData, err := ztpv6.ParseVendorData(packet); err != nil {
		glog.V(2).Infof("Failed to extract vendor data: %s", err)
	} else {
		message.Serial = vendorData.Serial
	}

	server, err := selectDestinationServer(s.config, &message)
	if err != nil {
		glog.Errorf("%s, Drop due to %s", packet.Summary(), err)
		s.logger.LogErr(start, nil, packet.ToBytes(), peer, ErrNoServer, err)
		return
	}

	relayMsg, err := dhcpv6.EncapsulateRelay(packet, dhcpv6.MessageTypeRelayForward, net.IPv6zero, peer.IP)
	sendToServer(s.logger, start, server, relayMsg.ToBytes(), peer, s.throttle)
}

func (s *serverImpl) handleV6RelayRepl(start time.Time, packet dhcpv6.DHCPv6, peer *net.UDPAddr) {
	// when we get a relay-reply, we need to unwind the message, removing the top
	// relay-reply info and passing on the inner part of the message
	msg, err := dhcpv6.DecapsulateRelay(packet)
	if err != nil {
		glog.Errorf("Failed to decapsulate packet, drop due to %s", err)
		s.logger.LogErr(start, nil, packet.ToBytes(), peer, ErrParse, err)
		return
	}
	peerAddr := packet.(*dhcpv6.DHCPv6Relay).PeerAddr()
	// send the packet to the peer addr
	addr := &net.UDPAddr{
		IP:   peerAddr,
		Port: dhcpv6.DefaultServerPort,
		Zone: "",
	}
	conn, err := net.DialUDP("udp", s.config.ReplyAddr, addr)
	if err != nil {
		glog.Errorf("Error creating udp connection %s", err)
		s.logger.LogErr(start, nil, packet.ToBytes(), peer, ErrConnect, err)
		return
	}
	conn.Write(msg.ToBytes())
	err = s.logger.LogSuccess(start, nil, packet.ToBytes(), peer)
	if err != nil {
		glog.Errorf("Failed to log request: %s", err)
	}
	conn.Close()
	return
}

func (s *serverImpl) handleV6Server(start time.Time, packet dhcpv6.DHCPv6, peer *net.UDPAddr) {
	reply, err := s.config.Handler.ServeDHCPv6(packet)
	logErr := s.logger.LogSuccess(start, nil, packet.ToBytes(), peer)
	if logErr != nil {
		glog.Errorf("Failed to log incoming packet: %s", logErr)
	}
	if err != nil {
		glog.Errorf("Error creating reply %s", err)
		s.logger.LogErr(start, nil, packet.ToBytes(), peer, fmt.Sprintf("%T", err), err)
		return
	}
	addr := &net.UDPAddr{
		IP:   peer.IP,
		Port: dhcpv6.DefaultServerPort,
	}
	s.conn.WriteTo(reply.ToBytes(), addr)
	err = s.logger.LogSuccess(start, nil, reply.ToBytes(), peer)
	if err != nil {
		glog.Errorf("Failed to log reply: %s", err)
	}
	return
}
