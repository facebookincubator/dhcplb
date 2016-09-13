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
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/golang/glog"
	"github.com/krolaw/dhcp4"
	"net"
	"runtime/debug"
	"sync"
	"time"
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

	// Check for connection rate per IP address
	ok, err := throttle.OK(peer.IP.String())
	if !ok {
		bufPool.Put(buffer)
		logger.LogErr(time.Now(), nil, nil, peer, ErrConnRate, err)
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
			handleRawPacketV4(logger, config, buffer[:bytesRead], peer)
		} else if config.Version == 6 {
			handleRawPacketV6(logger, config, buffer[:bytesRead], peer)
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

func selectDestinationServer(config *Config, message *DHCPMessage) (*DHCPServer, error) {
	server, err := handleOverride(config, message)
	if err != nil {
		glog.Errorf("Error handling override, drop due to: %s", err)
		return nil, err
	}
	if server == nil {
		server, err = config.Algorithm.selectRatioBasedDhcpServer(message)
	}
	return server, err
}

func handleOverride(config *Config, message *DHCPMessage) (*DHCPServer, error) {
	if override, ok := config.Overrides[FormatID(message.Mac)]; ok {
		glog.Infof("Found override rule for %s", FormatID(message.Mac))
		var server *DHCPServer
		var err error
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
	server, err := config.Algorithm.selectServerFromList(servers, message)
	if err != nil {
		return nil, fmt.Errorf("Failed to select server: %s", err)
	}
	return server, nil
}

func sendToServer(logger loggerHelper, start time.Time, server *DHCPServer, packet []byte, peer *net.UDPAddr) error {
	err := server.sendTo(packet)
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

func handleRawPacketV4(logger loggerHelper, config *Config, buffer []byte, peer *net.UDPAddr) {
	// runs in a separate go routine
	start := time.Now()
	var message DHCPMessage
	packet := dhcp4.Packet(buffer)

	message.XID = binary.BigEndian.Uint32(packet.XId())
	message.Peer = peer
	message.ClientID = packet.CHAddr()
	message.Mac = packet.CHAddr()
	t := dhcp4.MessageType(packet.ParseOptions()[dhcp4.OptionDHCPMessageType][0])
	packet.SetHops(packet.Hops() + 1)

	server, err := selectDestinationServer(config, &message)
	if err != nil {
		glog.Errorf("Xid: 0x%x, Type: %s, Hops+1: %x, GIAddr: %s, CHAddr: %s, "+
			"Drop due to %s", message.XID, t, packet.Hops(), packet.GIAddr(),
			FormatID(packet.CHAddr()), err)
		logger.LogErr(start, nil, packet, peer, ErrNoServer, err)
		return
	}

	sendToServer(logger, start, server, packet, peer)
}

func handleRawPacketV6(logger loggerHelper, config *Config, buffer []byte, peer *net.UDPAddr) {
	// runs in a separate go routine
	start := time.Now()
	packet := Packet6(buffer)

	t, err := packet.Type()
	if err != nil {
		glog.Errorf("Failed to get packet type: %s", err)
		return
	}
	if t == RelayRepl {
		handleV6RelayRepl(logger, start, packet, peer)
		return
	}

	var message DHCPMessage

	xid, err := packet.XID()
	if err != nil {
		glog.Errorf("Failed to extract XId, drop due to %s", err)
		logger.LogErr(start, nil, packet, peer, ErrParse, err)
		return
	}
	message.XID = xid
	message.Peer = peer
	duid, err := packet.Duid()
	if err != nil {
		glog.Errorf("Failed to extract DUID, drop due to %s", err)
		logger.LogErr(start, nil, packet, peer, ErrParse, err)
		return
	}
	message.ClientID = duid
	mac, err := packet.Mac()
	if err != nil {
		glog.Errorf("Failed to extract MAC, drop due to %s", err)
		logger.LogErr(start, nil, packet, peer, ErrParse, err)
		return
	}
	message.Mac = mac

	hops, _ := packet.Hops()
	link, _ := packet.LinkAddr()
	peerAddr, _ := packet.PeerAddr()

	server, err := selectDestinationServer(config, &message)
	if err != nil {
		glog.Errorf("Xid: 0x%x, Type: %s, Hops+1: %x, LinkAddr: %s, PeerAddr: %s, "+
			"DUID: %s, Drop due to %s", message.XID, t, hops, link, peerAddr, FormatID(duid), err)
		logger.LogErr(start, nil, packet, peer, ErrNoServer, err)
		return
	}

	relayMsg := packet.Encapsulate(peer.IP)
	sendToServer(logger, start, server, relayMsg, peer)
}

func handleV6RelayRepl(logger loggerHelper, start time.Time, packet Packet6, peer *net.UDPAddr) {
	// when we get a relay-reply, we need to unwind the message, removing the top
	// relay-reply info and passing on the inner part of the message
	msg, peerAddr, err := packet.Unwind()
	if err != nil {
		glog.Errorf("Failed to unwind packet, drop due to %s", err)
		logger.LogErr(start, nil, packet, peer, ErrParse, err)
		return
	}
	// send the packet to the peer addr
	addr := &net.UDPAddr{
		IP:   peerAddr,
		Port: int(547),
		Zone: "",
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		glog.Errorf("Error creating udp connection %s", err)
		logger.LogErr(start, nil, packet, peer, ErrConnect, err)
		return
	}
	conn.Write(msg)
	err = logger.LogSuccess(start, nil, packet, peer)
	if err != nil {
		glog.Errorf("Failed to log request: %s", err)
	}
	conn.Close()
	return
}
