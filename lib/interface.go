/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package dhcplb

import (
	"fmt"
	"net"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv6"
)

// DHCPMessage represents coordinates of a dhcp message.
type DHCPMessage struct {
	XID      fmt.Stringer
	Peer     *net.UDPAddr
	ClientID []byte
	Mac      net.HardwareAddr
	NetBoot  bool
	Serial   string
}

type id string

func (m *DHCPMessage) id() id {
	return id(fmt.Sprintf("%s%s%x", m.Peer.IP, m.XID, m.ClientID))
}

// DHCPBalancingAlgorithm defines an interface for load balancing algorithms.
// Users can implement their own and add them to config.go (in the
// configSpec.algorithm method)
type DHCPBalancingAlgorithm interface {
	SelectServerFromList(list []*DHCPServer, message *DHCPMessage) (*DHCPServer, error)
	SelectRatioBasedDhcpServer(message *DHCPMessage) (*DHCPServer, error)
	UpdateStableServerList(list []*DHCPServer) error
	UpdateRCServerList(list []*DHCPServer) error
	SetRCRatio(ratio uint32)
	// An unique name for the algorithm, this string can be used in the
	// configuration file, in the section where the algorithm is selecetd.
	Name() string
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

// Handler is an interface used while serving DHCP requests.
type Handler interface {
	ServeDHCPv4(packet *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, error)
	ServeDHCPv6(packet dhcpv6.DHCPv6) (dhcpv6.DHCPv6, error)
}
