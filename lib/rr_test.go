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
	"testing"
)

func RRTestEmpty(t *testing.T) {
	subject := new(roundRobin)
	_, err := subject.selectRatioBasedDhcpServer(&DHCPMessage{
		ClientID: []byte{0},
	})
	if err == nil {
		t.Fatalf("Should throw an error if server list is empty")
	}
}

func RRTestBalance(t *testing.T) {
	subject := new(roundRobin)
	servers := make([]*DHCPServer, 4)
	for i := 0; i < 4; i++ {
		servers[i] = &DHCPServer{
			Port: i,
		}
	}
	subject.updateStableServerList(servers)
	msg := DHCPMessage{
		ClientID: []byte{0},
	}
	for i := 0; i < 4; i++ {
		server, err := subject.selectRatioBasedDhcpServer(&msg)
		if err != nil {
			t.Fatalf("Unexpected error selecting server: %s", err)
		}
		if server.Port != i {
			t.Fatalf("Chose wrong server %d, expected %d", server.Port, i)
		}
	}
}
