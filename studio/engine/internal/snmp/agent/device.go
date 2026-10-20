package agent

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/proto"
)

// Emit receives the device's events (requests, writes, trap results) as worker event frames.
type Emit func(event map[string]any)

// Device is one running simulated agent.
type Device struct {
	cfg      model.Device
	conn     *net.UDPConn
	engineID []byte
	boots    int32
	now      func() time.Time
	emit     Emit
	log      *limiter

	mu      sync.Mutex // the fields below
	store   *store
	keys    map[string]*proto.Keys // by user name, localized to this engine
	users   map[string]model.User
	salt    uint64
	nextID  int32
	stats   map[string]uint64 // usmStats counters by OID
	pending map[int32]chan *proto.Message
	closed  bool
}

// Start binds the device's address and serves it until Close. boots is the engine boots
// value of this start; it must grow from one start to the next.
func Start(cfg model.Device, boots int32, now func() time.Time, emit Emit) (*Device, error) {
	if now == nil {
		now = time.Now
	}
	engineID, err := hex.DecodeString(cfg.EngineID)
	if err != nil || len(engineID) < 5 || len(engineID) > 32 {
		return nil, fmt.Errorf("%s: the engine ID must be 5 to 32 bytes of hex", cfg.Name)
	}
	points, err := newStore(cfg.Points, now())
	if err != nil {
		return nil, fmt.Errorf("%s: %v", cfg.Name, err)
	}
	d := &Device{cfg: cfg, engineID: engineID, boots: boots, now: now, emit: emit, log: newLimiter(now),
		store: points, keys: map[string]*proto.Keys{}, users: map[string]model.User{},
		stats: map[string]uint64{}, pending: map[int32]chan *proto.Message{}}
	cache := proto.KeyCache{}
	for _, user := range cfg.Users {
		keys, err := cache.Localize(user, engineID)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", cfg.Name, err)
		}
		d.keys[user.Name], d.users[user.Name] = keys, user
	}
	address, err := netip.ParseAddr(cfg.IP)
	if err != nil || !address.Is4() {
		return nil, fmt.Errorf("%s: %q is not an IPv4 address", cfg.Name, cfg.IP)
	}
	d.conn, err = net.ListenUDP("udp4", net.UDPAddrFromAddrPort(netip.AddrPortFrom(address, uint16(cfg.Port))))
	if err != nil {
		return nil, fmt.Errorf("%s: %s", cfg.Name, BindProblem(cfg.IP, cfg.Port, err))
	}
	go d.serve()
	return d, nil
}

// Close stops answering and releases the address.
func (d *Device) Close() {
	d.mu.Lock()
	d.closed = true
	d.mu.Unlock()
	d.conn.Close()
}

// clock is the engine boots and the seconds since this start.
func (d *Device) clock() (int32, int32) {
	return d.boots, int32(d.now().Sub(d.store.started) / time.Second)
}

func (d *Device) serve() {
	buffer := make([]byte, 65535)
	for {
		n, peer, err := d.conn.ReadFromUDPAddrPort(buffer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue // Windows reports ICMP port unreachable of an earlier send here
		}
		d.handle(append([]byte(nil), buffer[:n]...), peer)
	}
}

func (d *Device) send(packet []byte, peer netip.AddrPort) {
	if _, err := d.conn.WriteToUDPAddrPort(packet, peer); err != nil {
		d.request(peer, "", "", 0, "", "answer not sent: "+err.Error())
	}
}

func (d *Device) handle(packet []byte, peer netip.AddrPort) {
	m, err := proto.Decode(packet)
	if err != nil {
		d.request(peer, "", "", 0, "", "not a valid SNMP message")
		return
	}
	if m.PDU.Type == proto.Response || m.PDU.Type == proto.Report || (m.Header != nil && m.Header.Encrypted != nil &&
		d.awaiting(m)) {
		d.deliver(m)
		return
	}
	switch m.Version {
	case proto.Version1, proto.Version2c:
		d.handleCommunity(m, peer)
	case proto.Version3:
		if !d.cfg.V3 {
			d.request(peer, "v3", proto.PDUName(m.PDU.Type), 0, "", "ignored: SNMPv3 is off for this device")
			return
		}
		d.handleV3(m, peer)
	}
}

func (d *Device) handleCommunity(m *proto.Message, peer netip.AddrPort) {
	version := map[int]string{proto.Version1: "v1", proto.Version2c: "v2c"}[m.Version]
	op := proto.PDUName(m.PDU.Type)
	if (m.Version == proto.Version1 && !d.cfg.V1) || (m.Version == proto.Version2c && !d.cfg.V2c) {
		d.request(peer, version, op, len(m.PDU.VarBinds), "", "ignored: "+version+" is off for this device")
		return
	}
	readable := m.Community == d.cfg.ReadCommunity || m.Community == d.cfg.WriteCommunity
	if !readable || m.Community == "" {
		d.request(peer, version, op, len(m.PDU.VarBinds), "", "ignored: wrong community")
		return
	}
	if m.Version == proto.Version1 && m.PDU.Type == proto.GetBulkRequest {
		d.request(peer, version, op, len(m.PDU.VarBinds), "", "ignored: GETBULK needs v2c or v3")
		return
	}
	if !isRequest(m.PDU.Type) {
		return
	}
	d.mu.Lock()
	response, written := answer(d.store, m.PDU, m.Version, m.Community == d.cfg.WriteCommunity, d.now())
	d.mu.Unlock()
	packet, err := fitCommunity(m.Version, m.Community, response, m.PDU)
	if err != nil {
		d.request(peer, version, op, len(m.PDU.VarBinds), first(m.PDU), "answer not built: "+err.Error())
		return
	}
	d.send(packet, peer)
	d.request(peer, version, op, len(m.PDU.VarBinds), first(m.PDU), resultText(response))
	d.wrote(peer, written)
}

func isRequest(kind byte) bool {
	return kind == proto.GetRequest || kind == proto.GetNextRequest || kind == proto.GetBulkRequest ||
		kind == proto.SetRequest
}

// maxUDP is the largest answer sent (an IPv4 UDP payload).
const maxUDP = 65507

// fitCommunity encodes an answer; one too big for UDP becomes tooBig, and a GETBULK answer
// is cut to fit (RFC 3416 4.2.3).
func fitCommunity(version int, community string, response, request proto.PDU) ([]byte, error) {
	for {
		packet, err := proto.EncodeCommunity(version, community, response)
		if err != nil || len(packet) <= maxUDP {
			return packet, err
		}
		if request.Type == proto.GetBulkRequest && len(response.VarBinds) > 1 {
			response.VarBinds = response.VarBinds[:len(response.VarBinds)/2]
			continue
		}
		response.ErrorStatus, response.ErrorIndex, response.VarBinds = proto.TooBig, 0, []proto.VarBind{}
		return proto.EncodeCommunity(version, community, response)
	}
}

// Points is every point with its current value.
func (d *Device) Points() []model.Point {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.store.snapshot(d.now())
}

// SetValue changes a point from the page; SNMP managers see it at their next read.
func (d *Device) SetValue(oid string, value string, isHex bool) (model.Point, error) {
	parsed, err := proto.ParseOID(oid)
	if err != nil {
		return model.Point{}, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	item := d.store.get(parsed)
	if item == nil {
		return model.Point{}, fmt.Errorf("%s has no point %s", d.cfg.Name, oid)
	}
	next, err := proto.ParseValue(item.point.Type, value, isHex)
	if err != nil {
		return model.Point{}, err
	}
	d.store.set(item, next, d.now())
	return item.point, nil
}
