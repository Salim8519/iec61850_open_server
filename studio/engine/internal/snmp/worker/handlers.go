package worker

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"studio-engine/internal/snmp/agent"
	"studio-engine/internal/snmp/manager"
	"studio-engine/internal/snmp/model"
)

// Capability is what hello reports; the server refuses an older worker without it.
const Capability = "snmp_v1"

// handlers own the simulated devices, the reader session with its poller, and the trap receiver.
type handlers struct {
	emit   func(map[string]any)
	agents *agent.Engine

	mu       sync.Mutex // the reader and the receiver
	session  *manager.Session
	poller   *manager.Poller
	receiver *manager.Receiver
	closed   bool
}

func newHandlers(emit func(map[string]any)) *handlers {
	return &handlers{emit: emit, agents: agent.NewEngine(emit, nil)}
}

// close stops everything (stop request or end of input); it may run twice.
func (h *handlers) close() {
	h.agents.Stop()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	h.stopReader()
	if h.receiver != nil {
		h.receiver.Close()
		h.receiver = nil
	}
}

func decode[T any](r request) (T, error) {
	var params T
	if err := json.Unmarshal(r.Raw, &params); err != nil {
		return params, fmt.Errorf("invalid %s request: %v", r.Op, err)
	}
	return params, nil
}

func (h *handlers) dispatch(r request) (any, error) {
	if r.Version != 1 {
		return nil, fmt.Errorf("protocol version must be 1")
	}
	switch r.Op {
	case "hello":
		return map[string]any{"capabilities": []string{Capability}}, nil
	case "agents_start":
		params, err := decode[struct {
			Devices []model.Device `json:"devices"`
		}](r)
		if err != nil {
			return nil, err
		}
		return map[string]any{"started": h.agents.Start(params.Devices)}, nil
	case "agents_stop":
		h.agents.Stop()
		return map[string]any{}, nil
	case "agent_points":
		params, err := decode[deviceParams](r)
		if err != nil {
			return nil, err
		}
		points, err := h.agents.Points(params.Device)
		return map[string]any{"points": points}, err
	case "agent_set":
		params, err := decode[deviceParams](r)
		if err != nil {
			return nil, err
		}
		point, err := h.agents.SetValue(params.Device, params.OID, params.Value, params.Hex)
		return map[string]any{"point": point}, err
	case "agent_trap":
		params, err := decode[deviceParams](r)
		if err != nil {
			return nil, err
		}
		results, err := h.agents.SendTrap(params.Device, params.Trap)
		return map[string]any{"results": results}, err
	}
	return h.readerOp(r)
}

type deviceParams struct {
	Device string     `json:"device"`
	OID    string     `json:"oid"`
	Value  string     `json:"value"`
	Hex    bool       `json:"hex"`
	Trap   agent.Trap `json:"trap"`
}

type readerParams struct {
	Target  model.Target `json:"target"`
	Root    string       `json:"root"`
	OIDs    []string     `json:"oids"`
	EveryMS int          `json:"every_ms"`
	Point   model.Point  `json:"point"`
	IP      string       `json:"ip"`
	Port    int          `json:"port"`
	Users   []model.User `json:"users"`
	Engine  string       `json:"engine_id"`
}

func (h *handlers) readerOp(r request) (any, error) {
	params, err := decode[readerParams](r)
	if err != nil {
		return nil, err
	}
	switch r.Op {
	case "read_open":
		return h.open(params.Target)
	case "read_close":
		h.mu.Lock()
		h.stopReader()
		h.mu.Unlock()
		return map[string]any{}, nil
	case "read_walk":
		return h.walk(params.Root)
	case "read_get":
		return h.get(params.OIDs)
	case "read_set":
		return h.set(params.Point)
	case "read_poll":
		return h.poll(params.OIDs, time.Duration(params.EveryMS)*time.Millisecond)
	case "traps_listen":
		return h.listen(params)
	case "traps_stop":
		h.mu.Lock()
		if h.receiver != nil {
			h.receiver.Close()
			h.receiver = nil
		}
		h.mu.Unlock()
		return map[string]any{}, nil
	}
	return nil, fmt.Errorf("unknown operation %s", r.Op)
}
