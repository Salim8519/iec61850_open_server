package manager

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"studio-engine/internal/snmp/agent"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/proto"
)

var (
	sysUpTime = proto.MustOID("1.3.6.1.2.1.1.3.0")
	trapOID   = proto.MustOID("1.3.6.1.6.3.1.1.4.1.0")
	snmpTraps = proto.MustOID("1.3.6.1.6.3.1.1.5")
)

// Receiver listens for traps and informs; it acknowledges informs (v2c and v3) and is the
// authoritative engine for v3 informs.
type Receiver struct {
	conn     *net.UDPConn
	emit     func(map[string]any)
	engineID []byte
	boots    int32
	started  time.Time

	mu    sync.Mutex
	users map[string]model.User
	cache proto.KeyCache
	keys  map[string]*proto.Keys // by engine ID (hex) and user name
	salt  uint64
}

// Listen opens the trap port. users are the v3 users whose traps and informs are accepted.
func Listen(ip string, port int, users []model.User, engineID []byte, emit func(map[string]any)) (*Receiver, error) {
	address, err := netip.ParseAddr(ip)
	if err != nil || !address.Is4() {
		return nil, fmt.Errorf("%q is not an IPv4 address", ip)
	}
	r := &Receiver{emit: emit, engineID: engineID, started: time.Now(), users: map[string]model.User{},
		cache: proto.KeyCache{}, keys: map[string]*proto.Keys{},
		boots: int32(time.Since(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)) / time.Second)}
	for _, user := range users {
		if problem := proto.CheckUser(user); problem != "" {
			return nil, fmt.Errorf("%s", problem)
		}
		r.users[user.Name] = user
	}
	r.conn, err = net.ListenUDP("udp4", net.UDPAddrFromAddrPort(netip.AddrPortFrom(address, uint16(port))))
	if err != nil {
		return nil, fmt.Errorf("%s", agent.BindProblem(ip, port, err))
	}
	go r.serve()
	return r, nil
}

// Close stops listening.
func (r *Receiver) Close() { r.conn.Close() }

func (r *Receiver) serve() {
	buffer := make([]byte, 65535)
	for {
		n, peer, err := r.conn.ReadFromUDPAddrPort(buffer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
		r.handle(append([]byte(nil), buffer[:n]...), peer)
	}
}

func (r *Receiver) handle(packet []byte, peer netip.AddrPort) {
	m, err := proto.Decode(packet)
	if err != nil {
		r.report(peer, map[string]any{"problem": "not a valid SNMP message"})
		return
	}
	if m.Header != nil {
		r.handleV3(m, peer)
		return
	}
	version := map[int]string{proto.Version1: "v1", proto.Version2c: "v2c"}[m.Version]
	event := map[string]any{"version": version, "community": m.Community}
	switch m.PDU.Type {
	case proto.TrapV1:
		r.describeV1(event, m.PDU)
	case proto.TrapV2, proto.InformRequest:
		r.describe(event, m.PDU)
		if m.PDU.Type == proto.InformRequest {
			answer := proto.PDU{Type: proto.Response, RequestID: m.PDU.RequestID, VarBinds: m.PDU.VarBinds}
			if reply, err := proto.EncodeCommunity(m.Version, m.Community, answer); err == nil {
				r.conn.WriteToUDPAddrPort(reply, peer)
			}
		}
	default:
		return // requests to a trap port are not answered
	}
	r.report(peer, event)
}

// describeV1 maps a v1 trap to its notification OID (RFC 3584 3.1).
func (r *Receiver) describeV1(event map[string]any, pdu proto.PDU) {
	notification := append(append(proto.OID{}, pdu.Enterprise...), 0, uint32(pdu.SpecificTrap))
	if pdu.GenericTrap >= 0 && pdu.GenericTrap < 6 {
		notification = append(append(proto.OID{}, snmpTraps...), uint32(pdu.GenericTrap+1))
	}
	event["kind"] = "trap"
	event["notification"] = notification.String()
	event["uptime"] = int64(pdu.Timestamp)
	event["agent"] = netip.AddrFrom4(pdu.AgentAddr).String()
	event["varbinds"] = Points(pdu.VarBinds)
}

// describe reads sysUpTime and snmpTrapOID from a v2 notification.
func (r *Receiver) describe(event map[string]any, pdu proto.PDU) {
	event["kind"] = "trap"
	if pdu.Type == proto.InformRequest {
		event["kind"] = "inform"
	}
	rest := []proto.VarBind{}
	for _, bind := range pdu.VarBinds {
		switch {
		case bind.OID.Compare(sysUpTime) == 0:
			event["uptime"] = int64(bind.Value.Uint)
		case bind.OID.Compare(trapOID) == 0 && bind.Value.Type == proto.TagOID:
			event["notification"] = bind.Value.OID.String()
		default:
			rest = append(rest, bind)
		}
	}
	event["varbinds"] = Points(rest)
}

func (r *Receiver) report(peer netip.AddrPort, event map[string]any) {
	event["event"] = "trap"
	event["from"] = peer.String()
	if r.emit != nil {
		r.emit(event)
	}
}

// Points turns variable bindings into page rows.
func Points(binds []proto.VarBind) []any {
	rows := make([]any, len(binds))
	for index, bind := range binds {
		kind, text, isHex := proto.ValueText(bind.Value)
		rows[index] = map[string]any{"oid": bind.OID.String(), "type": kind, "value": text, "hex": isHex}
	}
	return rows
}

func engineKey(engineID []byte, user string) string { return hex.EncodeToString(engineID) + "/" + user }
