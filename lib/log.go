/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package dhcplb

import (
	"github.com/golang/glog"
	"net"
	"time"
)

// LogMessage holds the info of a log line.
type LogMessage struct {
	Version      int
	Packet       []byte
	Peer         *net.UDPAddr
	Server       string
	ServerIsRC   bool
	Latency      time.Duration
	Success      bool
	ErrorName    string
	ErrorDetails error
}

// PersonalizedLogger is an interface used to log a LogMessage using your own
// logic. It will be used in loggerHelperImpl.
type PersonalizedLogger interface {
	Log(msg LogMessage) error
}

// LoggerHelper is an interface used to log.
type loggerHelper interface {
	LogErr(start time.Time, server *DHCPServer, packet []byte, peer *net.UDPAddr, errName string, err error) error
	LogSuccess(start time.Time, server *DHCPServer, packet []byte, peer *net.UDPAddr) error
}

// loggerHelperImpl is the implementation of the above interface.
type loggerHelperImpl struct {
	personalizedLogger PersonalizedLogger
	version            int
}

func (h *loggerHelperImpl) LogErr(start time.Time, server *DHCPServer, packet []byte, peer *net.UDPAddr, errName string, err error) error {
	if h.personalizedLogger != nil {
		hostname := ""
		isRC := false
		if server != nil {
			hostname = server.Hostname
			isRC = server.IsRC
		}
		msg := LogMessage{
			Version:      h.version,
			Packet:       packet,
			Peer:         peer,
			Server:       hostname,
			ServerIsRC:   isRC,
			Latency:      time.Since(start),
			Success:      false,
			ErrorName:    errName,
			ErrorDetails: err,
		}
		err := h.personalizedLogger.Log(msg)
		if err != nil {
			glog.Errorf("Failed to log error: %s", err)
			return err
		}
	}
	return nil
}

func (h *loggerHelperImpl) LogSuccess(start time.Time, server *DHCPServer, packet []byte, peer *net.UDPAddr) error {
	if h.personalizedLogger != nil {
		hostname := ""
		isRC := false
		if server != nil {
			hostname = server.Hostname
			isRC = server.IsRC
		}
		msg := LogMessage{
			Version:    h.version,
			Packet:     packet,
			Peer:       peer,
			Server:     hostname,
			ServerIsRC: isRC,
			Latency:    time.Since(start),
			Success:    true,
		}
		err := h.personalizedLogger.Log(msg)
		if err != nil {
			glog.Errorf("Failed to log error: %s", err)
			return err
		}
	}
	return nil
}
