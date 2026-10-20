package manager

import (
	"fmt"
	"time"

	"studio-engine/internal/snmp/proto"
)

// v3State is what a non-authoritative engine knows about the device (RFC 3414 2.3).
type v3State struct {
	cache    proto.KeyCache
	engineID []byte
	boots    int32
	time     int32
	learned  time.Time
	keys     *proto.Keys
	salt     uint64
}

// discover learns the engine ID, boots and time (an empty-user request gets a report),
// then localizes the user's keys. An authenticated user also syncs the clock.
func (s *Session) discover() error {
	s.nextID++
	id := s.nextID
	packet, err := proto.EncodeV3(proto.Header{MsgID: id, Flags: proto.FlagReportable},
		proto.PDU{Type: proto.GetRequest, RequestID: id}, nil, 0)
	if err != nil {
		return err
	}
	reply, err := s.exchange(packet, func(m *proto.Message) bool { return m.Header != nil && m.Header.MsgID == id })
	if err != nil {
		return err
	}
	if len(reply.Header.EngineID) == 0 {
		return fmt.Errorf("%s answered engine discovery without an engine ID", s.peer)
	}
	state := s.v3
	state.engineID = append([]byte(nil), reply.Header.EngineID...)
	state.boots, state.time, state.learned = reply.Header.Boots, reply.Header.Time, time.Now()
	state.keys, err = state.cache.Localize(s.target.User, state.engineID)
	return err
}

// clock is the device's estimated boots and time now.
func (v *v3State) clock() (int32, int32) {
	return v.boots, v.time + int32(time.Since(v.learned)/time.Second)
}

// requestV3 sends a scoped PDU at the user's level and handles the time-window report once.
func (s *Session) requestV3(pdu proto.PDU) (proto.PDU, error) {
	state := s.v3
	for attempt := 0; attempt < 2; attempt++ {
		boots, clock := state.clock()
		state.salt++
		msgID := pdu.RequestID
		header := proto.Header{MsgID: msgID, MaxSize: 65507,
			Flags: proto.LevelFlags(s.target.User.Level) | proto.FlagReportable, EngineID: state.engineID,
			Boots: boots, Time: clock, UserName: s.target.User.Name, ContextEngineID: state.engineID,
			ContextName: s.target.Context}
		packet, err := proto.EncodeV3(header, pdu, state.keys, state.salt)
		if err != nil {
			return proto.PDU{}, err
		}
		reply, err := s.exchange(packet, func(m *proto.Message) bool { return m.Header != nil && m.Header.MsgID == msgID })
		if err != nil {
			return proto.PDU{}, err
		}
		if err := proto.Unprotect(reply, state.keys); err != nil {
			return proto.PDU{}, fmt.Errorf("the answer from %s did not pass the security check (%v): check both passwords "+
				"and protocols", s.peer, err)
		}
		if reply.PDU.Type != proto.Report {
			return reply.PDU, nil
		}
		if len(reply.PDU.VarBinds) == 0 {
			return proto.PDU{}, fmt.Errorf("%s sent an empty report", s.peer)
		}
		stat := reply.PDU.VarBinds[0].OID
		if stat.Compare(proto.StatsNotInTimeWindows) == 0 || stat.Compare(proto.StatsUnknownEngineIDs) == 0 {
			if len(reply.Header.EngineID) > 0 {
				state.engineID = append([]byte(nil), reply.Header.EngineID...)
				if state.keys, err = state.cache.Localize(s.target.User, state.engineID); err != nil {
					return proto.PDU{}, err
				}
			}
			state.boots, state.time, state.learned = reply.Header.Boots, reply.Header.Time, time.Now()
			s.nextID++
			pdu.RequestID = s.nextID
			continue
		}
		return proto.PDU{}, fmt.Errorf("%s refused user %s: %s", s.peer, s.target.User.Name, proto.ReportText(stat))
	}
	return proto.PDU{}, fmt.Errorf("the clock of %s could not be matched; try again", s.peer)
}
