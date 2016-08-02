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
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

// Packet6 contains raw bytes for a dhcpv6 packet
type Packet6 []byte

// MessageType represents various dhcpv6 message types
// go:generate stringer -type=MessageType packet6.go
type MessageType byte

// Various message types for DHCPv6
const (
	Solicit            MessageType = 1
	Advertise          MessageType = 2
	Request            MessageType = 3
	Confirm            MessageType = 4
	Renew              MessageType = 5
	Rebind             MessageType = 6
	Reply              MessageType = 7
	Release            MessageType = 8
	Decline            MessageType = 9
	Reconfigure        MessageType = 10
	InformationRequest MessageType = 11
	RelayForw          MessageType = 12
	RelayRepl          MessageType = 13
)

// OptionType represents various dhcpv6 option types
//go:generate stringer -type=OptionType packet6.go
type OptionType uint16

// List of Option Types
const (
	ClientID              OptionType = 1
	ServerID              OptionType = 2
	IdentAssocNonTempAddr OptionType = 3
	IdentAssocTempAddr    OptionType = 4
	IaAddr                OptionType = 5
	OptionRequest         OptionType = 6
	Preference            OptionType = 7
	ElapsedTime           OptionType = 8
	RelayMessage          OptionType = 9
	Auth                  OptionType = 11
	ServerUnicast         OptionType = 12
	StatusCode            OptionType = 13
	RapidCommit           OptionType = 14
	UserClass             OptionType = 15
	VendorClass           OptionType = 16
	VendorOpts            OptionType = 17
	InterfaceID           OptionType = 18
	ReconfigureMessage    OptionType = 19
	ReconfigureAccept     OptionType = 20
)

// DuidType is a uint16 integer, there can be 3 of them, see the enum below.
type DuidType uint16

// there are 3 types of DUIDs
const (
	DuidLLT DuidType = 1
	DuidEN  DuidType = 2
	DuidLL  DuidType = 3
)

func (p Packet6) getOption(option OptionType) ([]byte, error) {
	index := 4 // start of options are 4 bytes into a client/server message
	// start of options are 34 bytes into a Relay message
	if t := p.Type(); t == RelayForw || t == RelayRepl {
		index = 34
	}
	for index+4 < len(p) {
		optionType := OptionType(binary.BigEndian.Uint16(p[index : index+2]))
		optionLen := binary.BigEndian.Uint16(p[index+2 : index+4])
		if optionType == option {
			start := index + 4
			if start >= len(p) {
				return nil, fmt.Errorf("Found option %s, but start %d was out-of-bounds (%d)",
					option.String(), start, len(p))
			}
			if start+int(optionLen) > len(p) {
				return nil, fmt.Errorf("Found option %s, but end %d was out-of-bounds (%d)",
					option.String(), start+int(optionLen), len(p))
			}
			return p[start : start+int(optionLen)], nil
		}
		// else...
		index += 4 + int(optionLen) // skip ahead until the next option
	}
	return nil, fmt.Errorf("Failed to find option %s", option.String())
}

func (p Packet6) dhcp6message() (Packet6, error) {
	switch p.Type() {
	case RelayForw, RelayRepl:
		relayMsg, err := p.getOption(RelayMessage)
		if err != nil {
			return nil, err
		}
		return Packet6(relayMsg).dhcp6message()
	default:
		return p, nil
	}
}

// Type returns the MessageType for a Packet6
func (p Packet6) Type() MessageType { return MessageType(p[0]) }

// XID returns the Transaction ID for a Packet6
func (p Packet6) XID() (uint32, error) {
	msg, err := p.dhcp6message()
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(append([]byte{0}, msg[1:4]...)), nil
}

// Hops returns the number of hops for a Packet6
func (p Packet6) Hops() (byte, error) {
	if t := p.Type(); t != RelayForw && t != RelayRepl {
		return 0, errors.New("Not a RelayForw or RelayRepl, does not have hopcount")
	}
	return p[1], nil
}

// LinkAddr returns the LinkAddr field in the RelayInfo header. Will return
// error if the message is not a RelayForw or RelayRepl.
func (p Packet6) LinkAddr() (net.IP, error) {
	if t := p.Type(); t != RelayForw && t != RelayRepl {
		return nil, errors.New("Not a RelayForw or RelayRepl, does not have link-address")
	}
	return net.IP(p[2:18]), nil
}

// PeerAddr returns the PeerAddr field in the RelayInfo header. Will return
// error if the message is not a RelayForw or RelayRepl.
func (p Packet6) PeerAddr() (net.IP, error) {
	if t := p.Type(); t != RelayForw && t != RelayRepl {
		return nil, errors.New(
			"Not a RelayForw or RelayRepl, does not have peer-address")
	}
	return net.IP(p[18:34]), nil
}

// Duid returns the DUID field in Packet6
func (p Packet6) Duid() ([]byte, error) {
	m, err := p.dhcp6message()
	if err != nil {
		return nil, err
	}
	return m.getOption(ClientID)
}

// Mac returns the Mac addressed embededded in the DUID, note that thiw only
// works with type DuidLL and DuidLLT. An error will be returned otherwise.
func (p Packet6) Mac() ([]byte, error) {
	duid, err := p.Duid()
	if err != nil {
		return nil, err
	}
	duidType := DuidType(binary.BigEndian.Uint16(duid[0:2]))
	if duidType != DuidLLT && duidType != DuidLL {
		return nil, fmt.Errorf("Cannot extract MAC from DUID type %d", duidType)
	}
	// last 6 bytes of the duid will be the MAC address
	return duid[len(duid)-6:], nil
}

// Encapsulate embeds this message in a relay-forward message in preparation
// for forwarding to a relay/server
func (p Packet6) Encapsulate(peer net.IP) Packet6 {
	// 20.1 When a relay agent receives a
	// valid message to be relayed, it constructs a new Relay-forward
	// message.  The relay agent copies the source address from the header
	// of the IP datagram in which the message was received to the
	// peer-address field of the Relay-forward message.  The relay agent
	// copies the received DHCP message (excluding any IP or UDP headers)
	// into a Relay Message option in the new message.

	// create the relay-forward message
	// 1 byte message type, 1 byte hop count, 2x16 byte addresses,
	// 2 bytes for option code, 2 bytes for option length
	new := make([]byte, len(p)+2+2*16+2+2)
	new[0] = byte(RelayForw)
	hops, err := p.Hops()
	if err != nil {
		new[1] = 0
	} else {
		new[1] = hops + 1
	}

	// leave link-address empty
	copy(new[18:34], peer) // copy the peer address from the IP header

	optionCode := make([]byte, 2)
	binary.BigEndian.PutUint16(optionCode, uint16(RelayMessage))
	copy(new[34:36], optionCode)
	optionLen := make([]byte, 2)
	binary.BigEndian.PutUint16(optionLen, uint16(len(p)))
	copy(new[36:38], optionLen)

	copy(new[38:], p) // copy the whole message into the relay-message option
	return Packet6(new)
}

// Unwind strips off the relay-reply message from the outer Packet6 (p) to
// prepare for forwarding back to the client.
func (p Packet6) Unwind() (Packet6, net.IP, error) {
	// 20.2. Relaying a Relay-reply Message
	// The relay agent processes any options included in the Relay-reply
	// message in addition to the Relay Message option, and then discards
	// those options.
	// The relay agent extracts the message from the Relay Message option
	// and relays it to the address contained in the peer-address field of
	// the Relay-reply message.
	// If the Relay-reply message includes an Interface-id option, the relay
	// agent relays the message from the server to the client on the link
	// identified by the Interface-id option.  Otherwise, if the
	// link-address field is not set to zero, the relay agent relays the
	// message on the link identified by the link-address field.

	peer, err := p.PeerAddr()
	if err != nil {
		return nil, nil, err
	}

	relayMsg, err := p.getOption(RelayMessage)
	if err != nil {
		return nil, nil, errors.New("Failed to extract RelayMessage option")
	}

	return relayMsg, peer, nil
}
