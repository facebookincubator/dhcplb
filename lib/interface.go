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
	"fmt"
	"net"
)

// DHCPMessage represents coordinates of a dhcp message.
type DHCPMessage struct {
	XID      uint32
	Peer     *net.UDPAddr
	ClientID []byte
	Mac      []byte
}

type id string

func (m *DHCPMessage) id() id {
	return id(fmt.Sprintf("%s%d%x", m.Peer.IP, m.XID, m.ClientID))
}

type dhcpBalancingAlgorithm interface {
	selectServerFromList(list []*DHCPServer, message *DHCPMessage) (*DHCPServer, error)
	selectRatioBasedDhcpServer(message *DHCPMessage) (*DHCPServer, error)
	updateStableServerList(list []*DHCPServer) error
	updateRCServerList(list []*DHCPServer) error
	setRCRatio(ratio uint32)
}

// Server is the main interface implementing the DHCP server.
type Server interface {
	SetConfig(config *Config)
	ListenAndServe() error
	HasServers() bool
}

// DHCPServerSourcer is an interface used to fetch stable, rc and servers from
// a "tier" (group of servers).
type DHCPServerSourcer interface {
	GetStableServers() ([]*DHCPServer, error)
	GetRCServers() ([]*DHCPServer, error)
	GetServersFromTier(tier string) ([]*DHCPServer, error)
}

// Throttle is interface that implements rate limiting per key
type Throttle interface {
	// Returns whether the rate is below max for a key
	OK(interface{}) (bool, error)
	// Returns the number of items
	len() int
}
