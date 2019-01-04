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

type roundRobin struct {
	lock       sync.RWMutex
	stable     []*DHCPServer
	rc         []*DHCPServer
	rcRatio    uint32
	iterStable int
	iterRC     int
	iterList   int // iterator used by SelectServerFromList, can be used in stable/rc or passing list manually
}

func (rr *roundRobin) Name() string {
	return "rr"
}

func (rr *roundRobin) getHash(token []byte) uint32 {
	hasher := fnv.New32a()
	hasher.Write(token)
	hash := hasher.Sum32()
	return hash
}

func (rr *roundRobin) SetRCRatio(ratio uint32) {
	atomic.StoreUint32(&rr.rcRatio, ratio)
}

func (rr *roundRobin) SelectServerFromList(list []*DHCPServer, message *DHCPMessage) (*DHCPServer, error) {
	rr.lock.RLock()
	defer rr.lock.RUnlock()

	if len(list) == 0 {
		return nil, errors.New("Server list is empty")
	}
	// no guarantee that lists are the same size, so modulo before incrementing
	rr.iterList = rr.iterList % len(list)
	server := list[rr.iterList]
	rr.iterList++
	return server, nil
}

func (rr *roundRobin) SelectRatioBasedDhcpServer(message *DHCPMessage) (server *DHCPServer, err error) {
	// hash the clientid to see if it should be RC/Stable
	hash := rr.getHash(message.ClientID)

	rr.lock.Lock()
	defer rr.lock.Unlock()

	if hash%100 < rr.rcRatio {
		rr.iterList = rr.iterRC
		rr.iterRC++
		return rr.SelectServerFromList(rr.rc, message)
	}
	//otherwise go stable
	rr.iterList = rr.iterStable
	rr.iterStable++
	return rr.SelectServerFromList(rr.stable, message)
}

func (rr *roundRobin) UpdateServerList(name string, list []*DHCPServer, ptr *[]*DHCPServer) error {
	rr.lock.Lock()
	defer rr.lock.Unlock()

	*ptr = list
	rr.iterStable = 0
	rr.iterRC = 0
	glog.Infof("List of available %s servers:", name)
	for _, server := range *ptr {
		glog.Infof("%s", server)
	}
	return nil
}

func (rr *roundRobin) UpdateStableServerList(list []*DHCPServer) error {
	return rr.UpdateServerList("stable", list, &rr.stable)
}

func (rr *roundRobin) UpdateRCServerList(list []*DHCPServer) error {
	return rr.UpdateServerList("rc", list, &rr.rc)
}
