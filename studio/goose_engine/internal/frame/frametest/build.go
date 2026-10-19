// Package frametest builds GOOSE-shaped frames for tests of the routing packages.
package frametest

import "studio-goose-engine/internal/frame"

// Options describe one test frame. Zero values give an untagged frame for APPID 0x1000.
type Options struct {
	Dst     frame.MAC
	Src     frame.MAC
	AppID   uint16
	GocbRef string
	Tagged  bool
	Padding int
}

// GooseDst is a typical GOOSE multicast destination.
var GooseDst = frame.MAC{0x01, 0x0C, 0xCD, 0x01, 0x00, 0x01}

// Build returns a frame whose goosePdu starts with the gocbRef, like a real publisher's.
func Build(o Options) []byte {
	if o.Dst == (frame.MAC{}) {
		o.Dst = GooseDst
	}
	if o.AppID == 0 {
		o.AppID = 0x1000
	}
	ref := append([]byte{0x80}, berLength(len(o.GocbRef))...)
	ref = append(ref, o.GocbRef...)
	ref = append(ref, make([]byte, o.Padding)...)
	pdu := append([]byte{0x61}, berLength(len(ref))...)
	pdu = append(pdu, ref...)
	length := 8 + len(pdu)
	out := append([]byte{}, o.Dst[:]...)
	out = append(out, o.Src[:]...)
	if o.Tagged {
		out = append(out, 0x81, 0x00, 0x80, 0x00)
	}
	out = append(out, 0x88, 0xB8, byte(o.AppID>>8), byte(o.AppID), byte(length>>8), byte(length), 0, 0, 0, 0)
	return append(out, pdu...)
}

func berLength(n int) []byte {
	switch {
	case n < 0x80:
		return []byte{byte(n)}
	case n < 0x100:
		return []byte{0x81, byte(n)}
	default:
		return []byte{0x82, byte(n >> 8), byte(n)}
	}
}
