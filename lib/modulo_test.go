/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package dhcplb

import (
	"testing"
)

func Test_Empty(t *testing.T) {
	subject := new(modulo)
	_, err := subject.SelectRatioBasedDhcpServer(&DHCPMessage{
		ClientID: []byte{0},
	})
	if err == nil {
		t.Fatalf("Should throw an error if server list is empty")
	}
}

func Test_Hash(t *testing.T) {
	// these are randomly generated "client ids" that are known to result in
	// FNV-1a 32 bit hashes 0-4 after %4
	tests := [][]byte{
		[]byte{0xf6, 0x85, 0x63, 0x3, 0x11, 0x80, 0x72, 0x97, 0x23, 0xa1},
		[]byte{0x8c, 0x41, 0x34, 0xe1, 0x9c, 0xd, 0xfc, 0xe5, 0x41, 0x4b},
		[]byte{0x54, 0xc9, 0xeb, 0x57, 0xa, 0x57, 0x14, 0x43, 0x2b, 0x19},
		[]byte{0x54, 0xc5, 0x89, 0x66, 0xb2, 0xdc, 0x39, 0xf7, 0x8f, 0xa5},
	}
	subject := new(modulo)
	servers := make([]*DHCPServer, 4)
	for i := 0; i < 4; i++ {
		servers[i] = &DHCPServer{
			Port: i, //use port to tell if we picked the right one
		}
	}
	subject.UpdateStableServerList(servers)
	for i, v := range tests {
		msg := DHCPMessage{
			ClientID: v,
		}
		server, err := subject.SelectRatioBasedDhcpServer(&msg)
		if err != nil {
			t.Fatalf("Unexpected error selecting server: %s", err)
		}
		if server.Port != i {
			t.Fatalf("Chose wrong server for %x, was expecting %d, got %d",
				v, i, server.Port)
		}
	}
}
