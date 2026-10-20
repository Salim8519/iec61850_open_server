package agent

import (
	"fmt"
	"net/netip"
	"sync"
	"time"

	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/proto"
)

// Standard OIDs of a notification (RFC 3418).
var (
	sysUpTime  = proto.MustOID("1.3.6.1.2.1.1.3.0")
	trapOID    = proto.MustOID("1.3.6.1.6.3.1.1.4.1.0")
	snmpTraps  = proto.MustOID("1.3.6.1.6.3.1.1.5")
	sysObject  = proto.MustOID("1.3.6.1.2.1.1.2.0")
	informWait = 1500 * time.Millisecond
)

// informTries is how often an unanswered inform is sent.
const informTries = 3

// Trap is a notification the page asks a device to send to all its trap targets.
type Trap struct {
	Notification string        `json:"notification"` // dotted OID, e.g. linkDown 1.3.6.1.6.3.1.1.5.3
	VarBinds     []model.Point `json:"varbinds"`
}

// Result is what happened at one target.
type Result struct {
	Target string `json:"target"`
	OK     bool   `json:"ok"`
	Text   string `json:"text"`
}

// SendTrap sends trap to every target of the device and waits for the informs' answers.
func (d *Device) SendTrap(trap Trap) ([]Result, error) {
	notification, err := proto.ParseOID(trap.Notification)
	if err != nil {
		return nil, fmt.Errorf("notification: %v", err)
	}
	extra := make([]proto.VarBind, 0, len(trap.VarBinds))
	for _, point := range trap.VarBinds {
		oid, err := proto.ParseOID(point.OID)
		if err != nil {
			return nil, fmt.Errorf("value %s: %v", point.OID, err)
		}
		value, err := proto.ParseValue(point.Type, point.Value, point.Hex)
		if err != nil {
			return nil, fmt.Errorf("value %s: %v", point.OID, err)
		}
		extra = append(extra, proto.VarBind{OID: oid, Value: value})
	}
	if len(d.cfg.Traps) == 0 {
		return nil, fmt.Errorf("%s has no trap targets; add one under Traps", d.cfg.Name)
	}
	results := make([]Result, len(d.cfg.Traps))
	var wait sync.WaitGroup
	for index, target := range d.cfg.Traps {
		wait.Add(1)
		go func() {
			defer wait.Done()
			results[index] = d.sendTo(target, notification, extra)
		}()
	}
	wait.Wait()
	return results, nil
}

func (d *Device) sendTo(target model.TrapTarget, notification proto.OID, extra []proto.VarBind) Result {
	name := fmt.Sprintf("%s:%d (%s)", target.Host, target.Port, target.Version)
	address, err := netip.ParseAddr(target.Host)
	if err != nil || !address.Is4() {
		return Result{Target: name, Text: "the target address is not an IPv4 address"}
	}
	peer := netip.AddrPortFrom(address, uint16(target.Port))
	d.mu.Lock()
	uptime := d.store.uptime(d.now())
	d.nextID++
	id := d.nextID
	d.mu.Unlock()
	binds := append([]proto.VarBind{
		{OID: sysUpTime, Value: proto.Value{Type: proto.TagTimeTicks, Uint: uint64(uptime)}},
		{OID: trapOID, Value: proto.Value{Type: proto.TagOID, OID: notification}}}, extra...)
	kind := proto.TrapV2
	if target.Inform {
		kind = proto.InformRequest
	}
	pdu := proto.PDU{Type: kind, RequestID: id, VarBinds: binds}
	switch target.Version {
	case "v1":
		if target.Inform {
			return Result{Target: name, Text: "SNMPv1 has no informs; choose v2c or v3"}
		}
		packet, err := proto.EncodeCommunity(proto.Version1, target.Community, d.trapV1(notification, extra, uptime))
		return d.fire(name, packet, err, peer)
	case "v2c":
		packet, err := proto.EncodeCommunity(proto.Version2c, target.Community, pdu)
		if !target.Inform {
			return d.fire(name, packet, err, peer)
		}
		if err != nil {
			return Result{Target: name, Text: err.Error()}
		}
		return d.inform(name, peer, id, func() ([]byte, error) { return packet, nil })
	case "v3":
		return d.sendV3(name, peer, target, pdu)
	}
	return Result{Target: name, Text: "choose v1, v2c or v3"}
}

// trapV1 maps a v2 notification to a v1 trap (RFC 3584 3.2).
func (d *Device) trapV1(notification proto.OID, extra []proto.VarBind, uptime uint32) proto.PDU {
	trap := proto.PDU{Type: proto.TrapV1, VarBinds: extra, Timestamp: uptime, GenericTrap: 6}
	copy(trap.AgentAddr[:], netip.MustParseAddr(d.cfg.IP).AsSlice())
	last := notification[len(notification)-1]
	parent := notification[:len(notification)-1]
	switch {
	case parent.Compare(snmpTraps) == 0 && last >= 1 && last <= 6:
		trap.GenericTrap, trap.Enterprise = int(last)-1, d.enterprise()
	case len(parent) > 1 && parent[len(parent)-1] == 0:
		trap.Enterprise, trap.SpecificTrap = append(proto.OID{}, parent[:len(parent)-1]...), int(last)
	default:
		trap.Enterprise, trap.SpecificTrap = append(proto.OID{}, parent...), int(last)
	}
	return trap
}

// enterprise is the device's sysObjectID, or snmpTraps when it has none.
func (d *Device) enterprise() proto.OID {
	d.mu.Lock()
	defer d.mu.Unlock()
	if item := d.store.get(sysObject); item != nil && item.value.Type == proto.TagOID {
		return item.value.OID
	}
	return snmpTraps
}

func (d *Device) fire(name string, packet []byte, err error, peer netip.AddrPort) Result {
	if err != nil {
		return Result{Target: name, Text: err.Error()}
	}
	if _, err := d.conn.WriteToUDPAddrPort(packet, peer); err != nil {
		return Result{Target: name, Text: "not sent: " + err.Error()}
	}
	return Result{Target: name, OK: true, Text: "sent"}
}

// inform sends a message up to informTries times and waits for the answer with key.
func (d *Device) inform(name string, peer netip.AddrPort, key int32, build func() ([]byte, error)) Result {
	answer, err := d.exchange(peer, key, build)
	if err != nil {
		return Result{Target: name, Text: err.Error()}
	}
	if answer.PDU.Type == proto.Report && len(answer.PDU.VarBinds) > 0 {
		return Result{Target: name, Text: "refused: " + proto.ReportText(answer.PDU.VarBinds[0].OID)}
	}
	return Result{Target: name, OK: true, Text: "acknowledged"}
}

// exchange sends and waits for the message answering key (request-id, or msgID for v3).
func (d *Device) exchange(peer netip.AddrPort, key int32, build func() ([]byte, error)) (*proto.Message, error) {
	reply := make(chan *proto.Message, 1)
	d.mu.Lock()
	d.pending[key] = reply
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		delete(d.pending, key)
		d.mu.Unlock()
	}()
	for try := 0; try < informTries; try++ {
		packet, err := build()
		if err != nil {
			return nil, err
		}
		if _, err := d.conn.WriteToUDPAddrPort(packet, peer); err != nil {
			return nil, fmt.Errorf("not sent: %v", err)
		}
		select {
		case message := <-reply:
			return message, nil
		case <-time.After(informWait):
		}
	}
	return nil, fmt.Errorf("no answer after %d tries", informTries)
}

// awaiting reports an encrypted v3 message that answers one of our informs.
func (d *Device) awaiting(m *proto.Message) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, found := d.pending[m.Header.MsgID]
	return found
}

// deliver hands an answer to the inform waiting for it.
func (d *Device) deliver(m *proto.Message) {
	key := m.PDU.RequestID
	if m.Header != nil {
		key = m.Header.MsgID
	}
	d.mu.Lock()
	reply := d.pending[key]
	d.mu.Unlock()
	if reply != nil {
		select {
		case reply <- m:
		default:
		}
	}
}
