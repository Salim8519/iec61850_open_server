package agent

import (
	"bytes"
	"errors"
	"net/netip"

	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/proto"
)

var levelRank = map[string]int{model.LevelNoAuth: 0, model.LevelAuth: 1, model.LevelPrivacy: 2}

// handleV3 is the USM receive path of an authoritative engine (RFC 3414 3.2), then the request.
func (d *Device) handleV3(m *proto.Message, peer netip.AddrPort) {
	h := m.Header
	op := proto.PDUName(m.PDU.Type)
	if h.SecurityModel != proto.SecurityModelUSM {
		d.request(peer, "v3", op, 0, "", "ignored: unknown security model")
		return
	}
	if !bytes.Equal(h.EngineID, d.engineID) {
		// Discovery: an empty or foreign engine ID learns ours from the report.
		d.report(m, peer, proto.StatsUnknownEngineIDs, nil, "engine ID discovery from "+peer.Addr().String())
		return
	}
	d.mu.Lock()
	user, known := d.users[h.UserName]
	keys := d.keys[h.UserName]
	d.mu.Unlock()
	if !known {
		d.report(m, peer, proto.StatsUnknownUserNames, nil, "unknown user "+quote(h.UserName))
		return
	}
	asked := levelRank[h.Level()]
	if asked > levelRank[user.Level] {
		d.report(m, peer, proto.StatsUnsupportedSecLevels, nil, "user "+user.Name+" does not allow "+h.Level())
		return
	}
	if err := proto.Unprotect(m, keys); err != nil {
		if errors.Is(err, proto.ErrWrongDigest) {
			d.report(m, peer, proto.StatsWrongDigests, nil, "wrong authentication password for user "+user.Name)
		} else {
			d.report(m, peer, proto.StatsDecryptionErrors, nil, "wrong privacy password for user "+user.Name)
		}
		return
	}
	op = proto.PDUName(m.PDU.Type) // known only now when the request was encrypted
	boots, clock := d.clock()
	if h.Flags&proto.FlagAuth != 0 && (h.Boots != boots || abs(h.Time-clock) > proto.TimeWindow) {
		d.report(m, peer, proto.StatsNotInTimeWindows, keys, "time window sync with "+peer.Addr().String())
		return
	}
	if !isRequest(m.PDU.Type) {
		return
	}
	var response proto.PDU
	var written []*entry
	if asked < levelRank[user.Level] {
		// The user is configured for a higher level than this request used: no access.
		response = proto.PDU{Type: proto.Response, RequestID: m.PDU.RequestID, ErrorStatus: proto.AuthorizationError,
			VarBinds: m.PDU.VarBinds}
	} else {
		d.mu.Lock()
		response, written = answer(d.store, m.PDU, proto.Version3, true, d.now())
		d.mu.Unlock()
	}
	packet, err := d.encodeV3(h, response, keys, h.Flags&(proto.FlagAuth|proto.FlagPriv), m.PDU)
	if err != nil {
		d.request(peer, "v3", op, len(m.PDU.VarBinds), first(m.PDU), "answer not built: "+err.Error())
		return
	}
	d.send(packet, peer)
	d.request(peer, "v3 "+user.Name, op, len(m.PDU.VarBinds), first(m.PDU), resultText(response))
	d.wrote(peer, written)
}

// encodeV3 answers with this engine's ID, boots and time and the request's user and context.
// A GETBULK answer is cut to fit; any other answer too big for UDP becomes tooBig.
func (d *Device) encodeV3(request *proto.Header, response proto.PDU, keys *proto.Keys, flags byte,
	asked proto.PDU) ([]byte, error) {
	boots, clock := d.clock()
	header := proto.Header{MsgID: request.MsgID, MaxSize: maxUDP, Flags: flags, EngineID: d.engineID,
		Boots: boots, Time: clock, UserName: request.UserName, ContextEngineID: d.engineID,
		ContextName: request.ContextName}
	limit := maxUDP
	if request.MaxSize >= 484 && int(request.MaxSize) < limit {
		limit = int(request.MaxSize)
	}
	for {
		d.mu.Lock()
		d.salt++
		salt := d.salt
		d.mu.Unlock()
		packet, err := proto.EncodeV3(header, response, keys, salt)
		if err != nil || len(packet) <= limit {
			return packet, err
		}
		if asked.Type == proto.GetBulkRequest && len(response.VarBinds) > 1 {
			response.VarBinds = response.VarBinds[:len(response.VarBinds)/2]
			continue
		}
		response.ErrorStatus, response.ErrorIndex, response.VarBinds = proto.TooBig, 0, []proto.VarBind{}
		return proto.EncodeV3(header, response, keys, salt+1)
	}
}

// report answers a failed v3 message with the usmStats counter (RFC 3414 3.2), when the
// sender asked for reports. keys sign the report (only for the time window report).
func (d *Device) report(m *proto.Message, peer netip.AddrPort, stat proto.OID, keys *proto.Keys, why string) {
	d.mu.Lock()
	d.stats[stat.String()]++
	count := d.stats[stat.String()]
	d.mu.Unlock()
	d.request(peer, "v3", "report", 0, stat.String(), why)
	if m.Header.Flags&proto.FlagReportable == 0 {
		return
	}
	pdu := proto.PDU{Type: proto.Report, RequestID: m.PDU.RequestID,
		VarBinds: []proto.VarBind{{OID: stat, Value: proto.Value{Type: proto.TagCounter32, Uint: count & 0xffffffff}}}}
	flags := byte(0)
	if keys != nil {
		flags = proto.FlagAuth
	}
	packet, err := d.encodeV3(m.Header, pdu, keys, flags, proto.PDU{})
	if err == nil {
		d.send(packet, peer)
	}
}

func abs(value int32) int32 {
	if value < 0 {
		return -value
	}
	return value
}

func quote(text string) string { return "\"" + text + "\"" }
