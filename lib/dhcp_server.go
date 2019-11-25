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
)

// DHCPServer holds information about a single dhcp server
type DHCPServer struct {
	Hostname string
	Address  net.IP
	Port     int
	IsRC     bool
}

// NewDHCPServer returns an instance of DHCPServer
func NewDHCPServer(hostname string, ip net.IP, port int) *DHCPServer {
	s := DHCPServer{
		Hostname: hostname,
		Address:  ip,
		Port:     port,
	}
	return &s
}

func (d *DHCPServer) udpAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   d.Address,
		Port: d.Port,
		Zone: "",
	}
}

func (d *DHCPServer) String() string {
	if d.IsRC {
		return fmt.Sprintf("Hostname: %s, IP: %s, Port: %d (RC)", d.Hostname, d.Address, d.Port)
	}
	return fmt.Sprintf("Hostname: %s, IP: %s, Port: %d", d.Hostname, d.Address, d.Port)
}
