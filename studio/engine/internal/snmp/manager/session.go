// Package manager is ScadaBench's SNMP manager (ADR-0092): it reads and writes a device in
// v1, v2c or v3, walks it, polls chosen values and receives traps and informs.
package manager

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/proto"
)

// Session talks to one device; requests run one at a time.
type Session struct {
	target  model.Target
	peer    netip.AddrPort
	conn    *net.UDPConn
	timeout time.Duration
	retries int

	mu     sync.Mutex // one exchange at a time, and the fields below
	nextID int32
	v3     *v3State
	buffer []byte
}

// ErrTimeout is a device that did not answer.
var ErrTimeout = errors.New("no answer")

// Open prepares a session; v3 sessions learn the engine ID, boots and time at once.
func Open(target model.Target) (*Session, error) {
	address, err := netip.ParseAddr(target.Host)
	if err != nil || !address.Is4() {
		return nil, fmt.Errorf("%q is not an IPv4 address such as 192.168.1.20", target.Host)
	}
	if target.Port <= 0 || target.Port > 65535 {
		return nil, fmt.Errorf("the port must be 1 to 65535")
	}
	switch target.Version {
	case "v1", "v2c":
		if target.Community == "" {
			return nil, fmt.Errorf("type the community, for example public")
		}
	case "v3":
		if problem := proto.CheckUser(target.User); problem != "" {
			return nil, fmt.Errorf("%s", problem)
		}
	default:
		return nil, fmt.Errorf("choose v1, v2c or v3")
	}
	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, fmt.Errorf("cannot open a UDP socket: %v", err)
	}
	s := &Session{target: target, peer: netip.AddrPortFrom(address, uint16(target.Port)), conn: conn,
		timeout: time.Duration(max(target.TimeoutMS, 200)) * time.Millisecond, retries: max(target.Retries, 0),
		nextID: int32(time.Now().UnixNano() & 0x3fffffff), buffer: make([]byte, 65535)}
	if target.TimeoutMS == 0 {
		s.timeout = 2 * time.Second
	}
	if target.Version == "v3" {
		s.v3 = &v3State{cache: proto.KeyCache{}}
		s.mu.Lock()
		err = s.discover()
		s.mu.Unlock()
		if err != nil {
			conn.Close()
			return nil, err
		}
	}
	return s, nil
}

// Close releases the socket.
func (s *Session) Close() { s.conn.Close() }

// Target is what the session talks to.
func (s *Session) Target() model.Target { return s.target }

// EngineID is the device's v3 engine ID (nil for v1/v2c).
func (s *Session) EngineID() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.v3 == nil {
		return nil
	}
	return s.v3.engineID
}

func (s *Session) describe() string {
	if s.target.Version == "v3" {
		return fmt.Sprintf("%s (v3, user %s)", s.peer, s.target.User.Name)
	}
	return fmt.Sprintf("%s (%s, community %q)", s.peer, s.target.Version, s.target.Community)
}

// noAnswer explains a timeout.
func (s *Session) noAnswer() error {
	what := "the community"
	if s.target.Version == "v3" {
		what = "the user name and passwords"
	}
	return fmt.Errorf("%w from %s. Check the address, the port, %s and that SNMP is on in the device", ErrTimeout,
		s.describe(), what)
}

// Request sends one PDU and returns the device's response (errors in the PDU are the caller's).
func (s *Session) Request(pdu proto.PDU) (proto.PDU, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	pdu.RequestID = s.nextID
	if s.v3 != nil {
		return s.requestV3(pdu)
	}
	version := proto.Version2c
	if s.target.Version == "v1" {
		version = proto.Version1
	}
	packet, err := proto.EncodeCommunity(version, s.target.Community, pdu)
	if err != nil {
		return proto.PDU{}, err
	}
	reply, err := s.exchange(packet, func(m *proto.Message) bool {
		return m.Header == nil && m.PDU.Type == proto.Response && m.PDU.RequestID == pdu.RequestID
	})
	if err != nil {
		return proto.PDU{}, err
	}
	return reply.PDU, nil
}

// exchange sends packet (again after each timeout) until a message matching wanted arrives.
func (s *Session) exchange(packet []byte, wanted func(*proto.Message) bool) (*proto.Message, error) {
	for try := 0; try <= s.retries; try++ {
		if _, err := s.conn.WriteToUDPAddrPort(packet, s.peer); err != nil {
			return nil, fmt.Errorf("cannot send to %s: %v", s.peer, err)
		}
		deadline := time.Now().Add(s.timeout)
		for {
			s.conn.SetReadDeadline(deadline)
			n, from, err := s.conn.ReadFromUDPAddrPort(s.buffer)
			if err != nil {
				var timeout net.Error
				if errors.As(err, &timeout) && timeout.Timeout() {
					break
				}
				if errors.Is(err, net.ErrClosed) {
					return nil, fmt.Errorf("the session was closed")
				}
				continue // ICMP port unreachable from an earlier send: keep waiting
			}
			if from.Addr().Unmap() != s.peer.Addr() {
				continue
			}
			message, err := proto.Decode(append([]byte(nil), s.buffer[:n]...))
			if err == nil && wanted(message) {
				return message, nil
			}
		}
	}
	return nil, s.noAnswer()
}

// statusError words an error-status answer.
func statusError(pdu proto.PDU) error {
	if pdu.ErrorStatus == proto.NoError {
		return nil
	}
	names := map[int]string{proto.TooBig: "the answer is too big", proto.NoSuchName: "no such name",
		proto.BadValue: "bad value", proto.ReadOnly: "read-only", proto.GenErr: "general error",
		proto.NoAccess: "no access", proto.WrongType: "wrong type", proto.WrongLength: "wrong length",
		proto.WrongValue: "wrong value", proto.NoCreation: "the object does not exist",
		proto.NotWritable: "not writable", proto.AuthorizationError: "not authorized",
		proto.InconsistentValue: "inconsistent value", proto.ResourceUnavailable: "resource unavailable",
		proto.CommitFailed: "commit failed"}
	text := names[pdu.ErrorStatus]
	if text == "" {
		text = fmt.Sprintf("error %d", pdu.ErrorStatus)
	}
	if pdu.ErrorIndex > 0 && pdu.ErrorIndex <= len(pdu.VarBinds) {
		return fmt.Errorf("the device refused %s: %s", pdu.VarBinds[pdu.ErrorIndex-1].OID, text)
	}
	return fmt.Errorf("the device refused: %s", text)
}
