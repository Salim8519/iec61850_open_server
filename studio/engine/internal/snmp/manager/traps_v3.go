package manager

import (
	"bytes"
	"net/netip"
	"time"

	"studio-engine/internal/snmp/proto"
)

// handleV3 accepts a v3 trap (the sender is authoritative: keys are localized to its engine
// ID) or a v3 inform (this receiver is authoritative: discovery, time window, answer).
func (r *Receiver) handleV3(m *proto.Message, peer netip.AddrPort) {
	h := m.Header
	event := map[string]any{"version": "v3", "user": h.UserName}
	ours := bytes.Equal(h.EngineID, r.engineID)
	if len(h.EngineID) == 0 || (!ours && h.Flags&proto.FlagReportable != 0 && h.Flags&proto.FlagAuth == 0 &&
		h.UserName == "") {
		r.answerReport(m, peer, proto.StatsUnknownEngineIDs, nil) // an inform sender discovering us
		return
	}
	r.mu.Lock()
	user, known := r.users[h.UserName]
	r.mu.Unlock()
	if !known {
		event["problem"] = "unknown user " + h.UserName + ": add it under Traps to accept its messages"
		r.report(peer, event)
		if ours {
			r.answerReport(m, peer, proto.StatsUnknownUserNames, nil)
		}
		return
	}
	keys, err := r.keysFor(h.EngineID, h.UserName)
	if err == nil {
		err = proto.Unprotect(m, keys)
	}
	if err != nil {
		event["problem"] = "the message did not pass the security check for user " + user.Name +
			": check the passwords and protocols"
		r.report(peer, event)
		return
	}
	switch m.PDU.Type {
	case proto.TrapV2:
		r.describe(event, m.PDU)
	case proto.InformRequest:
		if !ours {
			return
		}
		boots, clock := r.clock()
		if h.Flags&proto.FlagAuth != 0 && (h.Boots != boots || abs(h.Time-clock) > proto.TimeWindow) {
			r.answerReport(m, peer, proto.StatsNotInTimeWindows, keys)
			return
		}
		r.describe(event, m.PDU)
		answer := proto.PDU{Type: proto.Response, RequestID: m.PDU.RequestID, VarBinds: m.PDU.VarBinds}
		r.answer(h, answer, keys, h.Flags&(proto.FlagAuth|proto.FlagPriv), peer)
	default:
		return
	}
	r.report(peer, event)
}

func (r *Receiver) keysFor(engineID []byte, name string) (*proto.Keys, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := engineKey(engineID, name)
	if keys := r.keys[key]; keys != nil {
		return keys, nil
	}
	keys, err := r.cache.Localize(r.users[name], engineID)
	if err == nil {
		r.keys[key] = keys
	}
	return keys, err
}

func (r *Receiver) clock() (int32, int32) {
	return r.boots, int32(time.Since(r.started) / time.Second)
}

func (r *Receiver) answerReport(m *proto.Message, peer netip.AddrPort, stat proto.OID, keys *proto.Keys) {
	if m.Header.Flags&proto.FlagReportable == 0 {
		return
	}
	pdu := proto.PDU{Type: proto.Report, RequestID: m.PDU.RequestID,
		VarBinds: []proto.VarBind{{OID: stat, Value: proto.Value{Type: proto.TagCounter32, Uint: 1}}}}
	flags := byte(0)
	if keys != nil {
		flags = proto.FlagAuth
	}
	r.answer(m.Header, pdu, keys, flags, peer)
}

func (r *Receiver) answer(request *proto.Header, pdu proto.PDU, keys *proto.Keys, flags byte, peer netip.AddrPort) {
	boots, clock := r.clock()
	r.mu.Lock()
	r.salt++
	salt := r.salt
	r.mu.Unlock()
	header := proto.Header{MsgID: request.MsgID, Flags: flags, EngineID: r.engineID, Boots: boots, Time: clock,
		UserName: request.UserName, ContextEngineID: request.ContextEngineID, ContextName: request.ContextName}
	if packet, err := proto.EncodeV3(header, pdu, keys, salt); err == nil {
		r.conn.WriteToUDPAddrPort(packet, peer)
	}
}

func abs(value int32) int32 {
	if value < 0 {
		return -value
	}
	return value
}
