// Package frame reads the few GOOSE header fields the engine routes on.
// It never decodes the dataset: studio_live (libiec61850) owns that.
package frame

import (
	"errors"
	"fmt"
)

// EtherTypeGoose is the IEC 61850-8-1 GOOSE EtherType.
const EtherTypeGoose = 0x88B8

// MaxFrame is the largest Ethernet frame the engine forwards (1518 with one VLAN tag).
const MaxFrame = 1518

// MAC is a 6-byte Ethernet address.
type MAC [6]byte

// String formats a MAC the way studio_live reports it: 01-0C-CD-01-00-01.
func (m MAC) String() string {
	return fmt.Sprintf("%02X-%02X-%02X-%02X-%02X-%02X", m[0], m[1], m[2], m[3], m[4], m[5])
}

// Multicast reports whether the group bit is set.
func (m MAC) Multicast() bool { return m[0]&1 == 1 }

// Info is what routing needs from one GOOSE frame.
type Info struct {
	Dst     MAC
	Src     MAC
	Tagged  bool
	AppID   uint16
	GocbRef string
	Size    int
}

var errNotGoose = errors.New("not a GOOSE frame")

// Parse reads the Ethernet header, optional 802.1Q tag, APPID and the gocbRef,
// which is the first element (tag 0x80) inside the goosePdu (tag 0x61).
func Parse(b []byte) (Info, error) {
	var info Info
	if len(b) < 26 {
		return info, errNotGoose
	}
	copy(info.Dst[:], b[0:6])
	copy(info.Src[:], b[6:12])
	pos := 12
	if b[pos] == 0x81 && b[pos+1] == 0x00 {
		info.Tagged = true
		pos += 4
	}
	if len(b) < pos+10 || int(b[pos])<<8|int(b[pos+1]) != EtherTypeGoose {
		return info, errNotGoose
	}
	pos += 2
	info.AppID = uint16(b[pos])<<8 | uint16(b[pos+1])
	pos += 8 // APPID, length, reserved1, reserved2
	if b[pos] != 0x61 {
		return info, errors.New("GOOSE frame without goosePdu")
	}
	pos++
	if _, n, err := berLength(b, pos); err != nil {
		return info, err
	} else {
		pos += n
	}
	if pos >= len(b) || b[pos] != 0x80 {
		return info, errors.New("GOOSE frame without gocbRef")
	}
	pos++
	length, n, err := berLength(b, pos)
	if err != nil {
		return info, err
	}
	pos += n
	if pos+length > len(b) {
		return info, errors.New("GOOSE gocbRef is truncated")
	}
	info.GocbRef = string(b[pos : pos+length])
	info.Size = len(b)
	return info, nil
}

// berLength decodes a BER definite length and returns it with the bytes it used.
func berLength(b []byte, pos int) (int, int, error) {
	if pos >= len(b) {
		return 0, 0, errors.New("GOOSE length is truncated")
	}
	first := b[pos]
	if first < 0x80 {
		return int(first), 1, nil
	}
	count := int(first & 0x7F)
	if count == 0 || count > 3 || pos+1+count > len(b) {
		return 0, 0, errors.New("GOOSE length is not supported")
	}
	value := 0
	for i := 0; i < count; i++ {
		value = value<<8 | int(b[pos+1+i])
	}
	return value, 1 + count, nil
}
