/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package dhcplb

import (
	"errors"
	"github.com/golang/glog"
	"hash/fnv"
	"sync"
	"sync/atomic"
)

type modulo struct {
	lock    sync.RWMutex
	stable  []*DHCPServer
	rc      []*DHCPServer
	rcRatio uint32
}

func (m *modulo) Name() string {
	return "xid"
}

func (m *modulo) getHash(token []byte) uint32 {
	hasher := fnv.New32a()
	hasher.Write(token)
	hash := hasher.Sum32()
	return hash
}

func (m *modulo) SetRCRatio(ratio uint32) {
	atomic.StoreUint32(&m.rcRatio, ratio)
}

func (m *modulo) SelectServerFromList(list []*DHCPServer, message *DHCPMessage) (*DHCPServer, error) {
	hash := m.getHash(message.ClientID)
	if len(list) == 0 {
		return nil, errors.New("Server list is empty")
	}
	return list[hash%uint32(len(list))], nil
}

func (m *modulo) SelectRatioBasedDhcpServer(message *DHCPMessage) (*DHCPServer, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	hash := m.getHash(message.ClientID)

	// convert to a number 0-100 and then see if it should be RC
	if hash%100 < m.rcRatio {
		return m.SelectServerFromList(m.rc, message)
	}
	// otherwise go to stable
	return m.SelectServerFromList(m.stable, message)
}

func (m *modulo) UpdateServerList(name string, list []*DHCPServer, ptr *[]*DHCPServer) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	*ptr = list
	glog.Infof("List of available %s servers:", name)
	for _, server := range *ptr {
		glog.Infof("%s", server)
	}
	return nil
}

func (m *modulo) UpdateStableServerList(list []*DHCPServer) error {
	return m.UpdateServerList("stable", list, &m.stable)
}

func (m *modulo) UpdateRCServerList(list []*DHCPServer) error {
	return m.UpdateServerList("rc", list, &m.rc)
}
