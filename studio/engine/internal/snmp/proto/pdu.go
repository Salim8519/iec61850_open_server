package proto

import "fmt"

// PDU types (RFC 3416, RFC 1157).
const (
	GetRequest     byte = 0xa0
	GetNextRequest byte = 0xa1
	Response       byte = 0xa2
	SetRequest     byte = 0xa3
	TrapV1         byte = 0xa4
	GetBulkRequest byte = 0xa5
	InformRequest  byte = 0xa6
	TrapV2         byte = 0xa7
	Report         byte = 0xa8
)

// Error status values (RFC 3416).
const (
	NoError             = 0
	TooBig              = 1
	NoSuchName          = 2
	BadValue            = 3
	ReadOnly            = 4
	GenErr              = 5
	NoAccess            = 6
	WrongType           = 7
	WrongLength         = 8
	WrongValue          = 10
	NoCreation          = 11
	InconsistentValue   = 12
	ResourceUnavailable = 13
	CommitFailed        = 14
	NotWritable         = 17
	AuthorizationError  = 16
)

// PDU is one protocol data unit. For GETBULK, ErrorStatus is non-repeaters and ErrorIndex
// max-repetitions. The v1 trap fields are used only when Type is TrapV1.
type PDU struct {
	Type        byte
	RequestID   int32
	ErrorStatus int
	ErrorIndex  int
	VarBinds    []VarBind

	Enterprise   OID
	AgentAddr    [4]byte
	GenericTrap  int
	SpecificTrap int
	Timestamp    uint32
}

// PDUName is a short readable name for logs.
func PDUName(kind byte) string {
	names := map[byte]string{GetRequest: "get", GetNextRequest: "get-next", Response: "response",
		SetRequest: "set", TrapV1: "trap", GetBulkRequest: "get-bulk", InformRequest: "inform",
		TrapV2: "trap", Report: "report"}
	if name, known := names[kind]; known {
		return name
	}
	return fmt.Sprintf("pdu 0x%02x", kind)
}

func encodeVarBinds(binds []VarBind) ([]byte, error) {
	items := make([][]byte, 0, len(binds))
	for _, bind := range binds {
		name, err := encodeOID(bind.OID)
		if err != nil {
			return nil, err
		}
		value, err := encodeValue(bind.Value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", bind.OID, err)
		}
		items = append(items, sequence(TagSequence, tlv(TagOID, name), value))
	}
	return sequence(TagSequence, items...), nil
}

// EncodePDU is the BER of one PDU.
func EncodePDU(p PDU) ([]byte, error) {
	binds, err := encodeVarBinds(p.VarBinds)
	if err != nil {
		return nil, err
	}
	if p.Type == TrapV1 {
		enterprise, err := encodeOID(p.Enterprise)
		if err != nil {
			return nil, err
		}
		return sequence(TrapV1, tlv(TagOID, enterprise), tlv(TagIPAddress, p.AgentAddr[:]),
			intElement(int64(p.GenericTrap)), intElement(int64(p.SpecificTrap)),
			tlv(TagTimeTicks, encodeUint(uint64(p.Timestamp))), binds), nil
	}
	return sequence(p.Type, intElement(int64(p.RequestID)), intElement(int64(p.ErrorStatus)),
		intElement(int64(p.ErrorIndex)), binds), nil
}

// decodePDU reads one PDU element.
func decodePDU(r *reader) (PDU, error) {
	tag, content, err := r.next()
	if err != nil {
		return PDU{}, err
	}
	if tag < GetRequest || tag > Report {
		return PDU{}, malformed(fmt.Sprintf("unknown PDU 0x%02x", tag))
	}
	p := PDU{Type: tag}
	body := &reader{data: content}
	if tag == TrapV1 {
		return decodeTrapV1(p, body)
	}
	id, err := body.integer("request-id")
	if err != nil {
		return p, err
	}
	status, err := body.integer("error-status")
	if err != nil {
		return p, err
	}
	index, err := body.integer("error-index")
	if err != nil {
		return p, err
	}
	p.RequestID, p.ErrorStatus, p.ErrorIndex = int32(id), int(status), int(index)
	p.VarBinds, err = decodeVarBinds(body)
	return p, err
}

func decodeTrapV1(p PDU, body *reader) (PDU, error) {
	content, err := body.expect(TagOID, "enterprise")
	if err != nil {
		return p, err
	}
	if p.Enterprise, err = decodeOID(content); err != nil {
		return p, err
	}
	address, err := body.expect(TagIPAddress, "agent-addr")
	if err != nil || len(address) != 4 {
		return p, malformed("agent-addr")
	}
	copy(p.AgentAddr[:], address)
	generic, err := body.integer("generic-trap")
	if err != nil {
		return p, err
	}
	specific, err := body.integer("specific-trap")
	if err != nil {
		return p, err
	}
	stamp, err := body.expect(TagTimeTicks, "time-stamp")
	if err != nil {
		return p, err
	}
	ticks, err := decodeUint(stamp)
	if err != nil {
		return p, err
	}
	p.GenericTrap, p.SpecificTrap, p.Timestamp = int(generic), int(specific), uint32(ticks)
	p.VarBinds, err = decodeVarBinds(body)
	return p, err
}

func decodeVarBinds(body *reader) ([]VarBind, error) {
	list, err := body.expect(TagSequence, "variable-bindings")
	if err != nil {
		return nil, err
	}
	items := &reader{data: list}
	binds := []VarBind{}
	for len(items.data) > 0 {
		content, err := items.expect(TagSequence, "varbind")
		if err != nil {
			return nil, err
		}
		pair := &reader{data: content}
		name, err := pair.expect(TagOID, "name")
		if err != nil {
			return nil, err
		}
		oid, err := decodeOID(name)
		if err != nil {
			return nil, err
		}
		tag, raw, err := pair.next()
		if err != nil {
			return nil, err
		}
		value, err := decodeValue(tag, raw)
		if err != nil {
			return nil, err
		}
		binds = append(binds, VarBind{OID: oid, Value: value})
	}
	return binds, nil
}
