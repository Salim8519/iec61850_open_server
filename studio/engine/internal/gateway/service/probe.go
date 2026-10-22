package service

import (
	"strconv"
	"sync"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
	"studio-engine/internal/tools/worker"
)

// ProbeTimeout: 10,000 addresses at 10 requests per second fit with margin.
const ProbeTimeout = 1500 * time.Second

// Probe is ModbusProbe: one bounded 'Find registers' at a time, in the background. Stop
// (Go only, change 0178) asks studio_master to end it after the request on the wire.
type Probe struct {
	mu      sync.Mutex
	current *pyjson.Object
	process worker.Process
}

// Snapshot is a copy of the probe state ({"state": "idle"} before the first probe).
func (p *Probe) Snapshot() *pyjson.Object {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil {
		return pyjson.NewObject().Set("state", "idle")
	}
	return copyObject(p.current)
}

// Running reports a probe in progress (the keeper skips Modbus polls meanwhile).
func (p *Probe) Running() bool { return get(p.Snapshot(), "state") == "running" }

// Start runs the native probe; done receives the final snapshot.
func (p *Probe) Start(process worker.Process, table string, start, end int64, done func(*pyjson.Object)) (*pyjson.Object, error) {
	p.mu.Lock()
	if p.current != nil && get(p.current, "state") == "running" {
		p.mu.Unlock()
		return nil, apperr.ValueError("Find registers is already running")
	}
	p.current = pyjson.NewObject().Set("state", "running").Set("table", table).Set("start", start).Set("end", end).
		Set("requests", int64(0)).Set("address", start).Set("found", []any{}).Set("message", "")
	p.process = process
	p.mu.Unlock()
	go p.run(process, table, start, end, done)
	return p.Snapshot(), nil
}

// Stop asks a running probe to end after the request it is waiting for; the ranges found so
// far stay. It answers the snapshot at once (the probe turns "stopped" when the worker answers).
func (p *Probe) Stop() (*pyjson.Object, error) {
	p.mu.Lock()
	running := p.current != nil && get(p.current, "state") == "running"
	process := p.process
	p.mu.Unlock()
	if running && process != nil {
		if _, err := process.Request("probe_stop", worker.DefaultTimeout, pyjson.NewObject()); err != nil {
			return nil, err
		}
	}
	return p.Snapshot(), nil
}

func (p *Probe) progress(event *pyjson.Object) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil || get(p.current, "state") != "running" {
		return
	}
	requests, found := event.Get("requests")
	if !found {
		requests = int64(0)
	}
	address, found := event.Get("address")
	if !found {
		address = int64(0)
	}
	p.current.Set("requests", requests).Set("address", address)
}

func (p *Probe) run(process worker.Process, table string, start, end int64, done func(*pyjson.Object)) {
	reply, err := process.Request("probe", ProbeTimeout, pyjson.NewObject().Set("table", table).Set("start", start).
		Set("end", end))
	answer, _ := reply.(*pyjson.Object)
	p.mu.Lock()
	if err != nil {
		p.current.Set("state", "failed").Set("message", err.Error())
	} else if get(answer, "stopped") == true {
		requests, _ := pytext.Int64(get(answer, "requests"))
		p.current.Set("state", "stopped").Set("found", get(answer, "found")).Set("requests", get(answer, "requests")).
			Set("message", "Stopped by you after "+strconv.FormatInt(requests, 10)+" requests.")
	} else {
		p.current.Set("state", "done").Set("found", get(answer, "found")).Set("requests", get(answer, "requests"))
	}
	p.mu.Unlock()
	done(p.Snapshot())
}
