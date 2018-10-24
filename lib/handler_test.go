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

	"github.com/google/go-cmp/cmp"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv6"
)

//SOLICIT message wrapped in Relay-Forw
var relayForwBytesDuidUUID = []byte{
	0x0c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0xfe, 0x80, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x26, 0x8a, 0x07, 0xff, 0xfe, 0x56,
	0xdc, 0xa4, 0x00, 0x12, 0x00, 0x06, 0x24, 0x8a,
	0x07, 0x56, 0xdc, 0xa4, 0x00, 0x09, 0x00, 0x5a,
	0x06, 0x7d, 0x9b, 0xca, 0x00, 0x01, 0x00, 0x12,
	0x00, 0x04, 0xb7, 0xfd, 0x0a, 0x8c, 0x1b, 0x14,
	0x10, 0xaa, 0xeb, 0x0a, 0x5b, 0x3f, 0xe8, 0x9d,
	0x0f, 0x56, 0x00, 0x06, 0x00, 0x0a, 0x00, 0x17,
	0x00, 0x18, 0x00, 0x17, 0x00, 0x18, 0x00, 0x01,
	0x00, 0x08, 0x00, 0x02, 0xff, 0xff, 0x00, 0x03,
	0x00, 0x28, 0x07, 0x56, 0xdc, 0xa4, 0x00, 0x00,
	0x0e, 0x10, 0x00, 0x00, 0x15, 0x18, 0x00, 0x05,
	0x00, 0x18, 0x26, 0x20, 0x01, 0x0d, 0xc0, 0x82,
	0x90, 0x63, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xaf, 0xa0, 0x00, 0x00, 0x1c, 0x20, 0x00, 0x00,
	0x1d, 0x4c}

func Test_Mac(t *testing.T) {
	packet, err := dhcpv6.FromBytes(relayForwBytesDuidUUID)
	if err != nil {
		t.Fatalf("Error encoding DHCPv6 packet: %s", err)
	}
	mac, errMac := Mac(packet)
	if errMac != nil {
		t.Fatalf("Error extracting mac from peer-address relayinfo: %s", errMac)
	}
	if FormatID(mac) != "24:8a:07:56:dc:a4" {
		t.Fatalf("Expected mac %s but got %s", "24:8a:07:56:de:b0", FormatID(mac))
	}
}

func TestParseV4VendorClass(t *testing.T) {
	tt := []struct {
		name         string
		vc, hostname string
		want         VendorData
		fail         bool
	}{
		{name: "empty", fail: true},
		{name: "unknownVendor", vc: "VendorX;BFR10K;XX12345", fail: true},
		{name: "truncatedVendor", vc: "Arista;1234", fail: true},
		{
			name: "arista",
			vc:   "Arista;DCS-7050S-64;01.23;JPE12345678",
			want: VendorData{
				VendorName: "Arista", Model: "DCS-7050S-64", Serial: "JPE12345678"},
		},
		{
			name: "juniper",
			vc:   "Juniper-ptx1000-DD123",
			want: VendorData{VendorName: "Juniper", Model: "ptx1000", Serial: "DD123"},
		},
		{
			name: "juniperModelDash",
			vc:   "Juniper-qfx10002-36q-DN817",
			want: VendorData{VendorName: "Juniper", Model: "qfx10002-36q", Serial: "DN817"},
		},
		{
			name:     "juniperHostnameSerial",
			vc:       "Juniper-qfx10008",
			hostname: "DE123",
			want:     VendorData{VendorName: "Juniper", Model: "qfx10008", Serial: "DE123"},
		},
		{
			name: "juniperNoSerial",
			vc:   "Juniper-qfx10008",
			want: VendorData{VendorName: "Juniper", Model: "qfx10008", Serial: ""},
		},
		{
			name: "juniperInvalid",
			vc:   "Juniper-",
			want: VendorData{VendorName: "Juniper", Model: "", Serial: ""},
		},
		{
			name: "juniperInvalid2",
			vc:   "Juniper-qfx99999-",
			want: VendorData{VendorName: "Juniper", Model: "qfx99999", Serial: ""},
		},
		{
			name: "zpe",
			vc:   "ZPESystems:NSC:001234567",
			want: VendorData{VendorName: "ZPESystems", Model: "NSC", Serial: "001234567"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			packet, err := dhcpv4.New()
			if err != nil {
				t.Fatalf("failed to creat dhcpv4 packet object: %v", err)
			}

			packet.AddOption(&dhcpv4.OptClassIdentifier{
				Identifier: tc.vc,
			})

			if tc.hostname != "" {
				packet.AddOption(&dhcpv4.OptHostName{
					HostName: tc.hostname,
				})
			}

			vd := VendorData{}

			if err := parseV4VendorClass(&vd, packet); err != nil && !tc.fail {
				t.Errorf("unexpected failure: %v", err)
			}

			if !cmp.Equal(tc.want, vd) {
				t.Errorf("unexpected VendorData:\n%s", cmp.Diff(tc.want, vd))
			}
		})
	}
}

func TestParseV4VIVC(t *testing.T) {
	tt := []struct {
		name  string
		entID uint32
		input []byte
		want  VendorData
		fail  bool
	}{
		{name: "empty", fail: true},
		{
			name:  "ciscoIOSXR",
			entID: 0x09,
			input: []byte("SN:0;PID:R-IOSXRV9000-CC"),
			want:  VendorData{VendorName: "Cisco Systems", Model: "R-IOSXRV9000-CC", Serial: "0"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			packet, err := dhcpv4.New()
			if err != nil {
				t.Fatalf("failed to creat dhcpv4 packet object: %v", err)
			}
			packet.AddOption(&dhcpv4.OptVIVC{
				Identifiers: []dhcpv4.VIVCIdentifier{
					{EntID: tc.entID, Data: tc.input},
				},
			})

			vd := VendorData{}

			if err := parseV4VIVC(&vd, packet); err != nil && !tc.fail {
				t.Errorf("unexpected failure: %v", err)
			}

			if !cmp.Equal(tc.want, vd) {
				t.Errorf("unexpected VendorData:\n%s", cmp.Diff(tc.want, vd))
			}
		})
	}
}
