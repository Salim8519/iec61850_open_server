package worker

import (
	"encoding/hex"
	"fmt"
	"time"

	"studio-engine/internal/snmp/manager"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/proto"
)

// systemGroup is read on connect: sysDescr, sysObjectID, sysUpTime, sysContact, sysName, sysLocation.
var systemGroup = []string{"1.3.6.1.2.1.1.1.0", "1.3.6.1.2.1.1.2.0", "1.3.6.1.2.1.1.3.0", "1.3.6.1.2.1.1.4.0",
	"1.3.6.1.2.1.1.5.0", "1.3.6.1.2.1.1.6.0"}

// minPoll is the fastest poll interval accepted.
const minPoll = 200 * time.Millisecond

// stopReader ends polling and the session; h.mu is held.
func (h *handlers) stopReader() {
	if h.poller != nil {
		h.poller.Stop()
		h.poller = nil
	}
	if h.session != nil {
		h.session.Close()
		h.session = nil
	}
}

func parseOIDs(texts []string) ([]proto.OID, error) {
	oids := make([]proto.OID, len(texts))
	for index, text := range texts {
		oid, err := proto.ParseOID(text)
		if err != nil {
			return nil, err
		}
		oids[index] = oid
	}
	return oids, nil
}

func (h *handlers) open(target model.Target) (any, error) {
	h.mu.Lock()
	h.stopReader()
	h.mu.Unlock()
	session, err := manager.Open(target)
	if err != nil {
		return nil, err
	}
	oids, _ := parseOIDs(systemGroup)
	system, err := session.GetMany(oids, len(oids))
	if err != nil {
		session.Close()
		return nil, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		session.Close()
		return nil, fmt.Errorf("the SNMP engine is stopping")
	}
	h.session = session
	return map[string]any{"engine_id": hex.EncodeToString(session.EngineID()), "system": manager.Points(system)}, nil
}

func (h *handlers) current() (*manager.Session, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.session == nil {
		return nil, fmt.Errorf("connect to a device first")
	}
	return h.session, nil
}

func (h *handlers) walk(root string) (any, error) {
	session, err := h.current()
	if err != nil {
		return nil, err
	}
	if root == "" {
		root = "1.3.6.1"
	}
	start, err := proto.ParseOID(root)
	if err != nil {
		return nil, err
	}
	reported := 0
	values, err := session.Walk(start, func(count int) {
		if count-reported >= 250 {
			reported = count
			h.emit(map[string]any{"event": "walk_progress", "count": count})
		}
	})
	problem := ""
	if err != nil {
		problem = err.Error()
	}
	return map[string]any{"values": manager.Points(values), "problem": problem}, nil
}

func (h *handlers) get(texts []string) (any, error) {
	session, err := h.current()
	if err != nil {
		return nil, err
	}
	oids, err := parseOIDs(texts)
	if err != nil {
		return nil, err
	}
	values, err := session.GetMany(oids, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"values": manager.Points(values)}, nil
}

func (h *handlers) set(point model.Point) (any, error) {
	session, err := h.current()
	if err != nil {
		return nil, err
	}
	oid, err := proto.ParseOID(point.OID)
	if err != nil {
		return nil, err
	}
	value, err := proto.ParseValue(point.Type, point.Value, point.Hex)
	if err != nil {
		return nil, err
	}
	values, err := session.Set([]proto.VarBind{{OID: oid, Value: value}})
	if err != nil {
		return nil, err
	}
	return map[string]any{"values": manager.Points(values)}, nil
}

// poll replaces the poller; no OIDs (or no session) just stops it.
func (h *handlers) poll(texts []string, every time.Duration) (any, error) {
	oids, err := parseOIDs(texts)
	if err != nil {
		return nil, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.poller != nil {
		h.poller.Stop()
		h.poller = nil
	}
	if len(oids) == 0 || h.session == nil {
		return map[string]any{"polling": false}, nil
	}
	h.poller = manager.StartPoller(h.session, oids, max(every, minPoll), func(values []any, problem string) {
		h.emit(map[string]any{"event": "read_values", "values": values, "problem": problem})
	})
	return map[string]any{"polling": true}, nil
}

func (h *handlers) listen(params readerParams) (any, error) {
	engine, err := hex.DecodeString(params.Engine)
	if err != nil || len(engine) < 5 {
		return nil, fmt.Errorf("the trap receiver needs an engine ID of at least 5 bytes")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.receiver != nil {
		h.receiver.Close()
		h.receiver = nil
	}
	receiver, err := manager.Listen(params.IP, params.Port, params.Users, engine, h.emit)
	if err != nil {
		return nil, err
	}
	h.receiver = receiver
	return map[string]any{"listening": true}, nil
}
