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

func (s *serverImpl) StartUpdatingServerList() {
	glog.Infof("Starting to update server list...")
	go s.updateServersContinuous()
}

func (s *serverImpl) updateServersContinuous() {
	for {
		config := s.getConfig()
		stable, err := config.HostSourcer.GetStableServers()
		if err != nil {
			glog.Error(err)
		}
		rc, err := config.HostSourcer.GetRCServers()
		if err != nil {
			glog.Error(err)
		}
		if err == nil {
			if len(stable) > 0 {
				s.handleUpdatedList(s.stableServers, stable)
				err = config.Algorithm.UpdateStableServerList(stable)
				if err != nil {
					glog.Errorf("Error updating stable server list: %s", err)
				} else {
					s.stableServers = stable
				}
			}
			if len(rc) > 0 {
				s.handleUpdatedList(s.rcServers, rc)
				err = config.Algorithm.UpdateRCServerList(rc)
				if err != nil {
					glog.Errorf("Error updating RC server list: %s", err)
				} else {
					s.rcServers = rc
				}
			}
		}

		<-time.NewTimer(config.ServerUpdateInterval).C
	}
}

func (s *serverImpl) handleUpdatedList(old, new []*DHCPServer) {
	config := s.getConfig()
	added, removed := diffServersList(old, new)
	if len(added) > 0 || len(removed) > 0 {
		glog.Info("Server list updated")
		// open connections in newly added servers
		makeConnections(old, new)
		// close connections in removed servers after waiting for a timeout
		// wait some time to be sure that there are no requests that will try
		// to send to a server that already had its connection closed
		time.AfterFunc(config.FreeConnTimeout, func() {
			for _, old := range removed {
				err := old.disconnect()
				if err != nil {
					glog.Errorf("Unable to close connection to %s", old)
				}
			}
		})
	} else {
		// always makeConnections even if there was no change, so that sockets don't leak
		makeConnections(old, new)
	}
}

type serverKey struct {
	// have to store address as string otherwise serverKey can't be used as map key
	Address string
	Port    int
}

func diffServersList(original, updated []*DHCPServer) (added, removed []*DHCPServer) {
	added = make([]*DHCPServer, 0)
	removed = make([]*DHCPServer, 0)

	// find servers that were not in original list
	originalMap := make(map[serverKey]bool)
	for _, s := range original {
		key := serverKey{
			s.Address.String(),
			s.Port,
		}
		originalMap[key] = true
	}
	for _, new := range updated {
		key := serverKey{
			new.Address.String(),
			new.Port,
		}
		if _, ok := originalMap[key]; !ok {
			added = append(added, new)
		}
	}

	// find servers that are no longer in the new list
	newMap := make(map[serverKey]bool)
	for _, s := range updated {
		key := serverKey{
			s.Address.String(),
			s.Port,
		}
		newMap[key] = true
	}
	for _, old := range original {
		key := serverKey{
			old.Address.String(),
			old.Port,
		}
		if _, ok := newMap[key]; !ok {
			removed = append(removed, old)
		}
	}

	return added, removed
}

// makeConnections takes an original list of servers, and a new one and either
// copies connections over from the old list or opens new connections
func makeConnections(original, updated []*DHCPServer) {
	originalMap := make(map[serverKey]*net.UDPConn)
	for _, s := range original {
		key := serverKey{
			s.Address.String(),
			s.Port,
		}
		originalMap[key] = s.conn
	}
	// now copy connections into the new list if needed
	for _, new := range updated {
		key := serverKey{
			new.Address.String(),
			new.Port,
		}
		if conn, ok := originalMap[key]; ok && conn != nil {
			new.conn = conn
		} else {
			err := new.connect()
			if err != nil {
				glog.Errorf("Unable to open socket to %s", new)
			}
		}
	}
}
