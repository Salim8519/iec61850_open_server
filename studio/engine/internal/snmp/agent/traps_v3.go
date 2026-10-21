package agent

import (
	"fmt"
	"net/netip"
	"time"

	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/proto"
)

// sendV3 sends a v3 trap (this device is the authoritative engine) or a v3 inform (the
// receiver is: its engine ID, boots and time are discovered first, RFC 3414 4).
func (d *Device) sendV3(name string, peer netip.AddrPort, target model.TrapTarget, pdu proto.PDU) Result {
	d.mu.Lock()
	user, known := d.users[target.User]
	keys := d.keys[target.User]
	d.mu.Unlock()
	if !known {
		return Result{Target: name, Text: fmt.Sprintf("the target uses user %q, which this device does not have", target.User)}
	}
	flags := proto.LevelFlags(user.Level)
	if !target.Inform {
		boots, clock := d.clock()
		header := proto.Header{MsgID: pdu.RequestID, Flags: flags, EngineID: d.engineID, Boots: boots, Time: clock,
			UserName: user.Name, ContextEngineID: d.engineID}
		packet, err := proto.EncodeV3(header, pdu, keys, d.nextSalt())
		return d.fire(name, packet, err, peer)
	}
	remote, err := d.discover(peer)
	if err != nil {
		return Result{Target: name, Text: err.Error()}
	}
	remoteKeys, err := proto.KeyCache{}.Localize(user, remote.EngineID)
	if err != nil {
		return Result{Target: name, Text: err.Error()}
	}
	for attempt := 0; attempt < 2; attempt++ {
		learned := time.Now()
		build := func() ([]byte, error) {
			header := proto.Header{MsgID: pdu.RequestID, Flags: flags | proto.FlagReportable, EngineID: remote.EngineID,
				Boots: remote.Boots, Time: remote.Time + int32(time.Since(learned)/time.Second), UserName: user.Name,
				ContextEngineID: d.engineID}
			return proto.EncodeV3(header, pdu, remoteKeys, d.nextSalt())
		}
		reply, err := d.exchange(peer, pdu.RequestID, build)
		if err != nil {
			return Result{Target: name, Text: err.Error()}
		}
		if err := proto.Unprotect(reply, remoteKeys); err != nil {
			return Result{Target: name, Text: "the answer did not pass the security check: " + err.Error()}
		}
		if reply.PDU.Type == proto.Report && len(reply.PDU.VarBinds) > 0 &&
			reply.PDU.VarBinds[0].OID.Compare(proto.StatsNotInTimeWindows) == 0 {
			remote = reply.Header // the receiver's current boots and time
			d.mu.Lock()
			d.nextID++
			pdu.RequestID = d.nextID
			d.mu.Unlock()
			continue
		}
		if reply.PDU.Type == proto.Report && len(reply.PDU.VarBinds) > 0 {
			return Result{Target: name, Text: "refused: " + proto.ReportText(reply.PDU.VarBinds[0].OID)}
		}
		return Result{Target: name, OK: true, Text: "acknowledged"}
	}
	return Result{Target: name, Text: "the receiver's clock could not be matched"}
}

// discover asks the receiver of an inform for its engine ID, boots and time.
func (d *Device) discover(peer netip.AddrPort) (*proto.Header, error) {
	d.mu.Lock()
	d.nextID++
	id := d.nextID
	d.mu.Unlock()
	build := func() ([]byte, error) {
		return proto.EncodeV3(proto.Header{MsgID: id, Flags: proto.FlagReportable},
			proto.PDU{Type: proto.GetRequest, RequestID: id}, nil, 0)
	}
	reply, err := d.exchange(peer, id, build)
	if err != nil {
		return nil, fmt.Errorf("engine discovery: %v", err)
	}
	if reply.Header == nil || len(reply.Header.EngineID) == 0 {
		return nil, fmt.Errorf("engine discovery: the receiver sent no engine ID")
	}
	return reply.Header, nil
}

func (d *Device) nextSalt() uint64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.salt++
	return d.salt
}
