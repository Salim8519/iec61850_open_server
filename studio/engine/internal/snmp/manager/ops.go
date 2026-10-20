package manager

import (
	"fmt"

	"studio-engine/internal/snmp/proto"
)

// MaxWalk is the most values one walk collects, so a looping device cannot fill memory.
const MaxWalk = 200000

func nulls(oids []proto.OID) []proto.VarBind {
	binds := make([]proto.VarBind, len(oids))
	for index, oid := range oids {
		binds[index] = proto.VarBind{OID: oid, Value: proto.Null}
	}
	return binds
}

// Get reads exact OIDs. Missing ones come back as exception values (v2c/v3) or, in v1,
// the whole request fails with noSuchName.
func (s *Session) Get(oids []proto.OID) ([]proto.VarBind, error) {
	reply, err := s.Request(proto.PDU{Type: proto.GetRequest, VarBinds: nulls(oids)})
	if err != nil {
		return nil, err
	}
	return reply.VarBinds, statusError(reply)
}

// Set writes values and returns what the device answered.
func (s *Session) Set(binds []proto.VarBind) ([]proto.VarBind, error) {
	reply, err := s.Request(proto.PDU{Type: proto.SetRequest, VarBinds: binds})
	if err != nil {
		return nil, err
	}
	return reply.VarBinds, statusError(reply)
}

// Walk reads every value under root in order (GETBULK in v2c/v3, GETNEXT in v1).
// progress, when set, hears the count so far.
func (s *Session) Walk(root proto.OID, progress func(int)) ([]proto.VarBind, error) {
	found := []proto.VarBind{}
	cursor := root
	repetitions := 25
	for len(found) < MaxWalk {
		var pdu proto.PDU
		if s.target.Version == "v1" {
			pdu = proto.PDU{Type: proto.GetNextRequest, VarBinds: nulls([]proto.OID{cursor})}
		} else {
			pdu = proto.PDU{Type: proto.GetBulkRequest, ErrorIndex: repetitions, VarBinds: nulls([]proto.OID{cursor})}
		}
		reply, err := s.Request(pdu)
		if err != nil {
			return found, err
		}
		if reply.ErrorStatus == proto.TooBig && repetitions > 1 {
			repetitions /= 2
			continue
		}
		if reply.ErrorStatus == proto.NoSuchName && s.target.Version == "v1" {
			return found, nil // the v1 end of the MIB
		}
		if err := statusError(reply); err != nil {
			return found, err
		}
		if len(reply.VarBinds) == 0 {
			return found, nil
		}
		for _, bind := range reply.VarBinds {
			if bind.Value.Type == proto.TagEndOfView || !bind.OID.HasPrefix(root) {
				return found, nil
			}
			if bind.OID.Compare(cursor) <= 0 {
				return found, fmt.Errorf("%s returned %s after %s: the device's OIDs are out of order, "+
					"so the walk stopped", s.peer, bind.OID, cursor)
			}
			if bind.Value.IsException() {
				continue
			}
			found = append(found, bind)
			cursor = bind.OID
		}
		if progress != nil {
			progress(len(found))
		}
	}
	return found, fmt.Errorf("the walk stopped after %d values", MaxWalk)
}

// GetMany reads OIDs in requests of at most chunk values. In v1 a chunk that fails with
// noSuchName is read one by one so one missing OID does not hide the others.
func (s *Session) GetMany(oids []proto.OID, chunk int) ([]proto.VarBind, error) {
	out := make([]proto.VarBind, 0, len(oids))
	for start := 0; start < len(oids); start += chunk {
		part := oids[start:min(start+chunk, len(oids))]
		reply, err := s.Request(proto.PDU{Type: proto.GetRequest, VarBinds: nulls(part)})
		if err != nil {
			return out, err
		}
		if reply.ErrorStatus == proto.NoError {
			out = append(out, reply.VarBinds...)
			continue
		}
		if reply.ErrorStatus == proto.TooBig && len(part) > 1 {
			half, err := s.GetMany(part, max(len(part)/2, 1))
			out = append(out, half...)
			if err != nil {
				return out, err
			}
			continue
		}
		if reply.ErrorStatus != proto.NoSuchName || len(part) == 1 {
			if reply.ErrorStatus == proto.NoSuchName {
				out = append(out, proto.VarBind{OID: part[0], Value: proto.NoSuchObject})
				continue
			}
			return out, statusError(reply)
		}
		single, err := s.GetMany(part, 1)
		out = append(out, single...)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}
