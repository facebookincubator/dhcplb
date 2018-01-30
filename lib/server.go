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
	"github.com/golang/glog"
	"net"
	"sync"
	"sync/atomic"
	"unsafe"
)

// UDP acceptor
type serverImpl struct {
	version       int
	conn          *net.UDPConn
	logger        loggerHelper
	config        *Config
	stableServers []*DHCPServer
	rcServers     []*DHCPServer
	bufPool       sync.Pool
	throttle      Throttle
}

// returns a pointer to the current config struct, so that if it does get changed while being used,
// it shouldn't affect the caller and this copy struct should be GC'ed when it falls out of scope
func (s *serverImpl) getConfig() *Config {
	return s.config
}

func (s *serverImpl) ListenAndServe() error {
	s.StartUpdatingServerList()

	glog.Infof("Started thrift server, processing DHCP requests...")

	for {
		handleConnection(s.conn, s.getConfig(), s.logger, &s.bufPool, s.throttle)
	}
}

func (s *serverImpl) SetConfig(config *Config) {
	glog.Infof("Updating server config")
	// update server list because Algorithm instance was recreated
	config.Algorithm.UpdateStableServerList(s.stableServers)
	config.Algorithm.UpdateRCServerList(s.rcServers)
	atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(&s.config)), unsafe.Pointer(config))
	glog.Infof("Updated server config")
}

func (s *serverImpl) HasServers() bool {
	return len(s.stableServers) > 0 || len(s.rcServers) > 0
}

// NewServer initialized a Server before returning it.
func NewServer(config *Config, version int, personalizedLogger PersonalizedLogger) (Server, error) {
	conn, err := net.ListenUDP("udp", config.Addr)
	if err != nil {
		return nil, err
	}

	// setup logger
	var loggerHelper = &loggerHelperImpl{
		version:            version,
		personalizedLogger: personalizedLogger,
	}

	server := &serverImpl{
		version: version,
		conn:    conn,
		logger:  loggerHelper,
		config:  config,
	}

	// pool to reuse packet buffers
	server.bufPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, server.getConfig().PacketBufSize)
		},
	}

	glog.Infof("Setting up throttle: Cache Size: %d - Cache Rate: %d - Request Rate: %d",
		config.CacheSize, config.CacheRate, config.Rate)
	throttle, err := NewThrottle(config.CacheSize, config.CacheRate, config.Rate)
	if err != nil {
		return nil, err
	}
	server.throttle = throttle

	return server, nil
}
