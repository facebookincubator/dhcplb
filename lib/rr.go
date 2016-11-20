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

func (rr *roundRobin) getHash(token []byte) uint32 {
	hasher := fnv.New32a()
	hasher.Write(token)
	hash := hasher.Sum32()
	return hash
}

func (rr *roundRobin) setRCRatio(ratio uint32) {
	atomic.StoreUint32(&rr.rcRatio, ratio)
}

func (rr *roundRobin) selectServerFromList(list []*DHCPServer, message *DHCPMessage) (*DHCPServer, error) {
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

func (rr *roundRobin) selectRatioBasedDhcpServer(message *DHCPMessage) (server *DHCPServer, err error) {
	// hash the clientid to see if it should be RC/Stable
	hash := rr.getHash(message.ClientID)

	rr.lock.Lock()
	defer rr.lock.Unlock()

	if hash%100 < rr.rcRatio {
		rr.iterList = rr.iterRC
		rr.iterRC++
		return rr.selectServerFromList(rr.rc, message)
	}
	//otherwise go stable
	rr.iterList = rr.iterStable
	rr.iterStable++
	return rr.selectServerFromList(rr.stable, message)
}

func (rr *roundRobin) updateServerList(name string, list []*DHCPServer, ptr *[]*DHCPServer) error {
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

func (rr *roundRobin) updateStableServerList(list []*DHCPServer) error {
	return rr.updateServerList("stable", list, &rr.stable)
}

func (rr *roundRobin) updateRCServerList(list []*DHCPServer) error {
	return rr.updateServerList("rc", list, &rr.rc)
}
