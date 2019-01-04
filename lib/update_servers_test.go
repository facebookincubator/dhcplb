/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package dhcplb

import (
	"github.com/facebookgo/ensure"
	"net"
	"testing"
)

func Test_Nil(t *testing.T) {
	added, removed := diffServersList(nil, nil)
	// added and removed lists should be empty
	ensure.DeepEqual(t, added, make([]*DHCPServer, 0))
	ensure.DeepEqual(t, removed, make([]*DHCPServer, 0))
}

func UpdateServerTestEmpty(t *testing.T) {
	added, removed := diffServersList(make([]*DHCPServer, 0), make([]*DHCPServer, 0))
	// added and removed lists should be empty
	ensure.DeepEqual(t, added, make([]*DHCPServer, 0))
	ensure.DeepEqual(t, removed, make([]*DHCPServer, 0))
}

func Test_Add(t *testing.T) {
	var original []*DHCPServer
	updated := []*DHCPServer{
		&DHCPServer{
			Address: net.ParseIP("1.2.3.4"),
			Port:    1,
		},
		&DHCPServer{
			Address: net.ParseIP("5.6.7.8"),
			Port:    2,
		},
	}
	added, removed := diffServersList(original, updated)
	// original list was empty, so added should just be the new list
	ensure.DeepEqual(t, added, updated)
	ensure.DeepEqual(t, removed, make([]*DHCPServer, 0))
}

func Test_Remove(t *testing.T) {
	original := []*DHCPServer{
		&DHCPServer{
			Address: net.ParseIP("1.2.3.4"),
			Port:    1,
		},
		&DHCPServer{
			Address: net.ParseIP("5.6.7.8"),
			Port:    2,
		},
	}
	var updated []*DHCPServer
	added, removed := diffServersList(original, updated)
	// new list is empty, so removed should just be the original list
	ensure.DeepEqual(t, removed, original)
	ensure.DeepEqual(t, added, make([]*DHCPServer, 0))
}

func Test_Add_Remove(t *testing.T) {
	original := []*DHCPServer{
		&DHCPServer{
			Address: net.ParseIP("1.2.3.4"),
			Port:    1,
		},
	}
	updated := []*DHCPServer{
		&DHCPServer{
			Address: net.ParseIP("5.6.7.8"),
			Port:    2,
		},
	}
	added, removed := diffServersList(original, updated)
	ensure.DeepEqual(t, added, updated)
	ensure.DeepEqual(t, removed, original)
}
