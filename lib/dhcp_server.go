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
		err := d.conn.Close()
		if err != nil {
			return err
		}
		d.conn = nil
	}
	return nil
}

func (d *DHCPServer) sendTo(packet []byte) error {
	if d.conn == nil {
		return fmt.Errorf("No connection open to %s", d)
	}
	_, err := d.conn.Write(packet)
	if err != nil {
		// if failed, try to re-open socket and try again once
		err = d.connect()
		if err != nil {
			return err
		}
		_, err := d.conn.Write(packet)
		if err != nil {
			return err
		}
	}
	return err
}

func (d *DHCPServer) String() string {
	if d.IsRC {
		return fmt.Sprintf("%s:%d (RC)", d.Hostname, d.Port)
	}
	return fmt.Sprintf("%s:%d", d.Hostname, d.Port)
}
