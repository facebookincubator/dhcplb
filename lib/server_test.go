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
	"github.com/facebookgo/ensure"
	"testing"
)

func Test_Format_Empty(t *testing.T) {
	if FormatID(nil) != "" {
		t.Fatalf("Expected nil array to yield empty string")
	}
	if FormatID([]byte{}) != "" {
		t.Fatalf("Expected empty array to yield empty string")
	}
}

func Test_Format_Normal(t *testing.T) {
	result := FormatID([]byte{0xfa})
	ensure.DeepEqual(t, result, "fa")
	result = FormatID([]byte{0xfa, 0xce})
	ensure.DeepEqual(t, result, "fa:ce")
	result = FormatID([]byte{0xfa, 0xce, 0x12, 0x34})
	ensure.DeepEqual(t, result, "fa:ce:12:34")
	result = FormatID([]byte{0x12, 0x34, 0x56, 0x78, 0x9a})
	ensure.DeepEqual(t, result, "12:34:56:78:9a")
}
