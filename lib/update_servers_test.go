/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package dhcplb

import (
	"fmt"
	"net"
	"reflect"
	"testing"
)

func TestDiffServerList(t *testing.T) {
	for i, tt := range []struct {
		original []*DHCPServer
		updated  []*DHCPServer
	}{
		{
			original: []*DHCPServer{},
			updated:  []*DHCPServer{},
		},
		{
			original: []*DHCPServer{},
			updated: []*DHCPServer{

				{
					Address: net.ParseIP("1.2.3.4"),
					Port:    1,
				},
				{
					Address: net.ParseIP("5.6.7.8"),
					Port:    2,
				},
			},
		},
		{
			original: []*DHCPServer{
				{
					Address: net.ParseIP("1.2.3.4"),
					Port:    1,
				},
				{
					Address: net.ParseIP("5.6.7.8"),
					Port:    2,
				},
			},
			updated: []*DHCPServer{},
		},
		{
			original: []*DHCPServer{
				{
					Address: net.ParseIP("1.2.3.4"),
					Port:    1,
				},
			},
			updated: []*DHCPServer{
				{
					Address: net.ParseIP("5.6.7.8"),
					Port:    2,
				},
			},
		},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			added, removed := diffServersList(tt.original, tt.updated)
			if !reflect.DeepEqual(added, tt.updated) {
				t.Errorf("added %v, updated %v", added, tt.updated)
			}
			if !reflect.DeepEqual(removed, tt.original) {
				t.Errorf("removed %v, original %v", removed, tt.original)
			}
		})
	}
}
