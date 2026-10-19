// Package bus is the app side of the engine: simulated IEDs send raw GOOSE frames to
// one UDP port on 127.0.0.1, and accepted frames are forwarded to studio_live's inbox.
package bus

import (
	"errors"
	"net"
	"sync/atomic"
)

// Origin is the first byte of every datagram sent to the studio_live inbox.
const (
	OriginApp  byte = 0 // published by a simulated IED in this app
	OriginWire byte = 1 // heard on the network adapter
)

// Bus owns the loopback socket. It binds 127.0.0.1 only, never a network address.
type Bus struct {
	conn  *net.UDPConn
	inbox atomic.Pointer[net.UDPAddr]
}

// Listen opens the bus on a free loopback port.
func Listen() (*Bus, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		return nil, err
	}
	_ = conn.SetReadBuffer(4 << 20)
	_ = conn.SetWriteBuffer(4 << 20)
	return &Bus{conn: conn}, nil
}

// Port is the bus port the workers publish to.
func (b *Bus) Port() int { return b.conn.LocalAddr().(*net.UDPAddr).Port }

// SetInbox sets the studio_live port that receives accepted frames.
func (b *Bus) SetInbox(port int) error {
	if port <= 0 || port > 65535 {
		return errors.New("inbox port is out of range")
	}
	b.inbox.Store(&net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
	return nil
}

// Deliver forwards one frame to the inbox. It reports false when no inbox is set yet.
func (b *Bus) Deliver(origin byte, frame []byte) bool {
	inbox := b.inbox.Load()
	if inbox == nil {
		return false
	}
	datagram := make([]byte, 1+len(frame))
	datagram[0] = origin
	copy(datagram[1:], frame)
	_, err := b.conn.WriteToUDP(datagram, inbox)
	return err == nil
}

// Run reads worker frames until Close. Each frame is a fresh slice.
func (b *Bus) Run(onFrame func([]byte)) {
	buffer := make([]byte, 65536)
	for {
		n, _, err := b.conn.ReadFromUDP(buffer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue // a vanished peer can surface as a read error on Windows
		}
		onFrame(append([]byte(nil), buffer[:n]...))
	}
}

// Close stops Run and frees the port.
func (b *Bus) Close() { _ = b.conn.Close() }
