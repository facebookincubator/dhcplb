/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package dhcplb

import (
	"fmt"
	"github.com/golang/glog"
	"net"
	"sync"
)

// DHCPServer holds information about a single dhcp server
type DHCPServer struct {
	Hostname string
	Address  net.IP
	Port     int
	conn     *net.UDPConn
	connLock sync.Mutex
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

func (d *DHCPServer) connect() error {
	d.connLock.Lock()
	defer d.connLock.Unlock()
	if d.conn == nil {
		glog.Infof("Opening connection to %s", d)
		addr := &net.UDPAddr{
			IP:   d.Address,
			Port: d.Port,
			Zone: "",
		}
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			glog.Errorf("Failed to open connection to %s", d)
			return err
		}
		d.conn = conn
	}
	return nil
}

func (d *DHCPServer) disconnect() error {
	d.connLock.Lock()
	defer d.connLock.Unlock()
	if d.conn != nil {
		glog.Infof("Closing connection to %s", d)
		if err := d.conn.Close(); err != nil {
			return err
		}
		d.conn = nil
	}
	return nil
}

func (d *DHCPServer) sendTo(packet []byte) error {
	if d.conn == nil {
		glog.Errorf("No connection open to %s.", d)
		if err := d.connect(); err != nil {
			return err
		}
	}
	if _, err := d.conn.Write(packet); err != nil {
		// if failed, try to re-open socket and try again once
		if err := d.connect(); err != nil {
			return err
		}
		if _, err := d.conn.Write(packet); err != nil {
			return err
		}
	}
	return nil
}

func (d *DHCPServer) String() string {
	if d.IsRC {
		return fmt.Sprintf("Hostname: %s, IP: %s, Port: %d (RC)", d.Hostname, d.Address, d.Port)
	}
	return fmt.Sprintf("Hostname: %s, IP: %s, Port: %d", d.Hostname, d.Address, d.Port)
}
