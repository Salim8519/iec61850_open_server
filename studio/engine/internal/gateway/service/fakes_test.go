package service

import (
	"sync"

	"studio-engine/internal/pyjson"
)

// fakeProjects holds IED models and the gateway section in memory (stream B's role).
type fakeProjects struct {
	mu      sync.Mutex
	ieds    []*pyjson.Object
	section *pyjson.Object
	saves   int
}

func (p *fakeProjects) IEDs() []*pyjson.Object {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ieds
}

func (p *fakeProjects) Gateway() *pyjson.Object {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.section
}

func (p *fakeProjects) SaveGateway(config *pyjson.Object) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.section, p.saves = config, p.saves+1
	return nil
}

// fakeHub is the runtime and its value hub (stream C's role): values by (ied, ref).
type fakeHub struct {
	mu          sync.Mutex
	state       string
	signals     map[[2]string]Signal
	subscribers map[int]func(Change)
	next        int
	events      []string
	endpoint    [2]any
}

func newHub() *fakeHub {
	return &fakeHub{state: "running", signals: map[[2]string]Signal{}, subscribers: map[int]func(Change){},
		endpoint: [2]any{"127.65.0.40", int64(20102)}}
}

func (h *fakeHub) RunState() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.state
}

func (h *fakeHub) Signal(iedID, ref string) (Signal, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	signal, found := h.signals[[2]string{iedID, ref}]
	return signal, found
}

func (h *fakeHub) Endpoint(string) (string, any, error) {
	return h.endpoint[0].(string), h.endpoint[1], nil
}

func (h *fakeHub) Emit(kind string, _ *pyjson.Object) {
	h.mu.Lock()
	h.events = append(h.events, kind)
	h.mu.Unlock()
}

func (h *fakeHub) Subscribe(notify func(Change)) func() {
	h.mu.Lock()
	id := h.next
	h.next++
	h.subscribers[id] = notify
	h.mu.Unlock()
	return func() {
		h.mu.Lock()
		delete(h.subscribers, id)
		h.mu.Unlock()
	}
}

// set stores a value and notifies the subscribers, as the value hub does on a change.
func (h *fakeHub) set(iedID, ref string, value any) {
	h.mu.Lock()
	signal := h.signals[[2]string{iedID, ref}]
	signal.Value = value
	h.signals[[2]string{iedID, ref}] = signal
	notify := make([]func(Change), 0, len(h.subscribers))
	for _, subscriber := range h.subscribers {
		notify = append(notify, subscriber)
	}
	h.mu.Unlock()
	for _, subscriber := range notify {
		subscriber(Change{IEDID: iedID, Ref: ref})
	}
}

func (h *fakeHub) define(iedID, ref string, signal Signal) {
	h.mu.Lock()
	h.signals[[2]string{iedID, ref}] = signal
	h.mu.Unlock()
}

// fakeOperator plays the IED: an operate sets the status in the hub after a short delay.
type fakeOperator struct {
	hub    *fakeHub
	status string
	calls  []string
	mu     sync.Mutex
}

func (o *fakeOperator) Operate(host string, port any, reference string, value any, options any) (*pyjson.Object, error) {
	o.mu.Lock()
	o.calls = append(o.calls, reference)
	o.mu.Unlock()
	position := int64(1)
	if value == true {
		position = 2
	}
	go o.hub.set("bay", o.status, position)
	return pyjson.NewObject().Set("ok", true).Set("model", "sbo-with-enhanced-security").Set("selected", true), nil
}
