package agent

import (
	"time"

	"studio-engine/internal/snmp/proto"
)

// maxRepetitions caps one GETBULK so a request cannot ask for an unbounded answer.
const maxRepetitions = 200

// answer runs one request PDU against the store. v1 reports a missing OID as noSuchName;
// v2c and v3 use the exception values. canWrite says whether this community/user may SET.
// written lists the points a SET changed.
func answer(s *store, request proto.PDU, version int, canWrite bool, now time.Time) (response proto.PDU, written []*entry) {
	response = proto.PDU{Type: proto.Response, RequestID: request.RequestID}
	switch request.Type {
	case proto.GetRequest:
		response.VarBinds, response.ErrorStatus, response.ErrorIndex = get(s, request.VarBinds, version, now)
	case proto.GetNextRequest:
		response.VarBinds, response.ErrorStatus, response.ErrorIndex = getNext(s, request.VarBinds, version, now)
	case proto.GetBulkRequest:
		response.VarBinds = bulk(s, request, now)
	case proto.SetRequest:
		response.VarBinds, response.ErrorStatus, response.ErrorIndex, written = set(s, request.VarBinds, version, canWrite, now)
	}
	if response.ErrorStatus != proto.NoError {
		response.VarBinds = request.VarBinds // an error answer echoes the request (RFC 3416 4.2.1)
	}
	return response, written
}

func get(s *store, binds []proto.VarBind, version int, now time.Time) ([]proto.VarBind, int, int) {
	out := make([]proto.VarBind, len(binds))
	for index, bind := range binds {
		item := s.get(bind.OID)
		if item == nil {
			if version == proto.Version1 {
				return nil, proto.NoSuchName, index + 1
			}
			out[index] = proto.VarBind{OID: bind.OID, Value: s.missing(bind.OID)}
			continue
		}
		value := s.current(item, now)
		if version == proto.Version1 && value.Type == proto.TagCounter64 {
			return nil, proto.NoSuchName, index + 1 // v1 has no Counter64 (RFC 2576 4.2.2.1)
		}
		out[index] = proto.VarBind{OID: bind.OID, Value: value}
	}
	return out, proto.NoError, 0
}

func getNext(s *store, binds []proto.VarBind, version int, now time.Time) ([]proto.VarBind, int, int) {
	out := make([]proto.VarBind, len(binds))
	for index, bind := range binds {
		item := nextVisible(s, bind.OID, version)
		if item == nil {
			if version == proto.Version1 {
				return nil, proto.NoSuchName, index + 1
			}
			out[index] = proto.VarBind{OID: bind.OID, Value: proto.EndOfMibView}
			continue
		}
		out[index] = proto.VarBind{OID: item.oid, Value: s.current(item, now)}
	}
	return out, proto.NoError, 0
}

// nextVisible skips Counter64 points for v1 managers.
func nextVisible(s *store, oid proto.OID, version int) *entry {
	item := s.next(oid)
	for item != nil && version == proto.Version1 && item.value.Type == proto.TagCounter64 {
		item = s.next(item.oid)
	}
	return item
}

// bulk is RFC 3416 4.2.3: the first non-repeaters get one successor each, the rest
// max-repetitions successors each, row by row.
func bulk(s *store, request proto.PDU, now time.Time) []proto.VarBind {
	nonRepeaters := min(max(request.ErrorStatus, 0), len(request.VarBinds))
	repetitions := min(max(request.ErrorIndex, 0), maxRepetitions)
	out := []proto.VarBind{}
	for _, bind := range request.VarBinds[:nonRepeaters] {
		out = append(out, successor(s, bind.OID, now))
	}
	cursors := make([]proto.OID, 0, len(request.VarBinds)-nonRepeaters)
	for _, bind := range request.VarBinds[nonRepeaters:] {
		cursors = append(cursors, bind.OID)
	}
	for round := 0; round < repetitions && len(cursors) > 0; round++ {
		finished := true
		for column, cursor := range cursors {
			found := successor(s, cursor, now)
			out = append(out, found)
			if found.Value.Type != proto.TagEndOfView {
				cursors[column] = found.OID
				finished = false
			}
		}
		if finished {
			break
		}
	}
	return out
}

func successor(s *store, oid proto.OID, now time.Time) proto.VarBind {
	item := s.next(oid)
	if item == nil {
		return proto.VarBind{OID: oid, Value: proto.EndOfMibView}
	}
	return proto.VarBind{OID: item.oid, Value: s.current(item, now)}
}

// set checks every binding first and changes nothing unless all are acceptable.
func set(s *store, binds []proto.VarBind, version int, canWrite bool, now time.Time) ([]proto.VarBind, int, int, []*entry) {
	refuse := func(v2Status, index int) ([]proto.VarBind, int, int, []*entry) {
		if version == proto.Version1 {
			switch v2Status {
			case proto.WrongType, proto.WrongValue, proto.WrongLength:
				return nil, proto.BadValue, index, nil
			}
			return nil, proto.NoSuchName, index, nil // noAccess, notWritable, noCreation (RFC 2576 4.3)
		}
		return nil, v2Status, index, nil
	}
	items := make([]*entry, len(binds))
	for index, bind := range binds {
		if !canWrite {
			return refuse(proto.NoAccess, index+1)
		}
		item := s.get(bind.OID)
		if item == nil {
			return refuse(proto.NoCreation, index+1)
		}
		if !item.point.Writable {
			return refuse(proto.NotWritable, index+1)
		}
		if bind.Value.Type != item.value.Type {
			return refuse(proto.WrongType, index+1)
		}
		if bind.Value.Type == proto.TagIPAddress && len(bind.Value.Bytes) != 4 {
			return refuse(proto.WrongLength, index+1)
		}
		items[index] = item
	}
	for index, item := range items {
		s.set(item, binds[index].Value, now)
	}
	return binds, proto.NoError, 0, items
}
