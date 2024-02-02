/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package dhcplb

import (
	"context"
	"net"
	"sync/atomic"
	"unsafe"

	"github.com/golang/glog"
)

// UDP acceptor
type Server struct {
	server        bool
	conn          *net.UDPConn
	logger        *loggerHelper
	config        *Config
	stableServers []*DHCPServer
	rcServers     []*DHCPServer
	throttle      *Throttle
}

// returns a pointer to the current config struct, so that if it does get changed while being used,
// it shouldn't affect the caller and this copy struct should be GC'ed when it falls out of scope
func (s *Server) GetConfig() *Config {
	return s.config
}

// ListenAndServe starts the server
func (s *Server) ListenAndServe(ctx context.Context) error {
	if !s.server {
		s.startUpdatingServerList()
	}

	glog.Infof("Started server, processing DHCP requests...")

	for {
		s.handleConnection(ctx)
	}
}

// SetConfig updates the server config
func (s *Server) SetConfig(config *Config) {
	glog.Infof("Updating server config")
	// update server list because Algorithm instance was recreated
	config.Algorithm.UpdateStableServerList(s.stableServers)
	config.Algorithm.UpdateRCServerList(s.rcServers)
	atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(&s.config)), unsafe.Pointer(config))
	// update the throttle rate
	s.throttle.setRate(config.Rate)
	glog.Infof("Updated server config")
}

// HasServers checks if the list of backend servers is not empty
func (s *Server) HasServers() bool {
	return len(s.stableServers) > 0 || len(s.rcServers) > 0
}

// NewServer initialized a Server before returning it.
func NewServer(config *Config, serverMode bool, personalizedLogger PersonalizedLogger) (*Server, error) {
	conn, err := net.ListenUDP("udp", config.Addr)
	if err != nil {
		return nil, err
	}

	// setup logger
	var loggerHelper = &loggerHelper{
		version:            config.Version,
		personalizedLogger: personalizedLogger,
	}

	server := &Server{
		server: serverMode,
		conn:   conn,
		logger: loggerHelper,
		config: config,
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
