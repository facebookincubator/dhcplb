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
	"errors"
	"github.com/golang/glog"
	"hash/fnv"
	"sync"
)

type modulo struct {
	lock    sync.RWMutex
	stable  []*DHCPServer
	rc      []*DHCPServer
	rcRatio uint32
}

func (m *modulo) setRCRatio(ratio uint32) {
	m.rcRatio = ratio
}

func (m *modulo) selectServerFromList(list []*DHCPServer, message *DHCPMessage) (*DHCPServer, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if len(list) == 0 {
		return nil, errors.New("Server list is empty")
	}
	hasher := fnv.New32a()
	hasher.Write(message.ClientID)
	hash := hasher.Sum32()
	return list[hash%uint32(len(list))], nil
}

func (m *modulo) selectRatioBasedDhcpServer(message *DHCPMessage) (*DHCPServer, error) {
	hasher := fnv.New32a()
	hasher.Write(message.ClientID)
	hash := hasher.Sum32()

	// convert to a number 0-100 and then see if it should be RC
	if hash%100 < m.rcRatio {
		return m.selectServerFromList(m.rc, message)
	}
	// otherwise go to stable
	return m.selectServerFromList(m.stable, message)
}

func (m *modulo) updateServerList(name string, list []*DHCPServer, ptr *[]*DHCPServer) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	*ptr = list
	glog.Infof("List of available %s servers:", name)
	for _, server := range *ptr {
		glog.Infof("%s", server)
	}
	return nil
}

func (m *modulo) updateStableServerList(list []*DHCPServer) error {
	return m.updateServerList("stable", list, &m.stable)
}

func (m *modulo) updateRCServerList(list []*DHCPServer) error {
	return m.updateServerList("RC", list, &m.rc)
}
