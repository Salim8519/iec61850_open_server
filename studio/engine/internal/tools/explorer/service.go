package explorer

import (
	"errors"
	"net/netip"
	"os"
	"sync"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/control"
	"studio-engine/internal/tools/memory"
	"studio-engine/internal/tools/pytext"
	"studio-engine/internal/tools/worker"
)

// Watch and poll limits (tools/explorer.py).
const (
	MaxWatch      = 200
	MinIntervalMS = 500
	MaxIntervalMS = 60000
)

// Emitter publishes a bus event (kind plus payload keys in call order).
type Emitter func(kind string, payload *pyjson.Object)

// Service is ExplorerService: one warm studio_live worker that browses any IEC 61850
// server, polls a watch list and sends confirmed commands through the control client.
type Service struct {
	executable string
	spawn      worker.Spawner
	emit       Emitter
	memory     *memory.Store
	control    control.Operator
	changes    // OnChange: the push stream's "explorer" topic

	spawnMu   sync.Mutex // one worker start at a time
	connectMu sync.Mutex // one connect at a time
	mu        sync.Mutex // the fields below; never held across a worker request
	process   worker.Process
	host      string
	port      any // nil while no endpoint
	state     string
	message   string
	cache     map[string]any
	watch     []*pyjson.Object
	values    map[string]*pyjson.Object
	interval  int64
	polling   bool
	pollStop  chan struct{}
	pollDone  chan struct{}
	reads     int64
}

// NewService returns a disconnected Explorer for studio_live at executable.
func NewService(executable string, spawn worker.Spawner, emit Emitter, store *memory.Store, operator control.Operator) *Service {
	if spawn == nil {
		spawn = worker.Spawn
	}
	if emit == nil {
		emit = func(string, *pyjson.Object) {}
	}
	return &Service{executable: executable, spawn: spawn, emit: emit, memory: store, control: operator,
		state: "disconnected", cache: map[string]any{}, values: map[string]*pyjson.Object{}, interval: 1000,
		pollStop: make(chan struct{})}
}

// CheckEndpoint is check_endpoint: a dotted IPv4 address and a port from 1 to 65535.
func CheckEndpoint(host, port any) (string, any, error) {
	address, err := netip.ParseAddr(pytext.Strip(pytext.Str(host)))
	if err != nil || !address.Is4() || !strictIPv4(pytext.Strip(pytext.Str(host))) {
		return "", nil, apperr.ValueError("Type the device address as four numbers, for example 192.168.1.10")
	}
	if number, ok := pytext.Int64(port); !ok || !pytext.IsIntOrBool(port) || number < 1 || number > 65535 {
		return "", nil, apperr.ValueError("The port must be a number between 1 and 65535 (IEC 61850 uses 102)")
	}
	return address.String(), port, nil
}

// strictIPv4: four decimal octets without leading zeros, as ipaddress.IPv4Address wants.
func strictIPv4(text string) bool {
	parts, start := 0, 0
	for index := 0; index <= len(text); index++ {
		if index == len(text) || text[index] == '.' {
			octet := text[start:index]
			if octet == "" || len(octet) > 3 || (len(octet) > 1 && octet[0] == '0') {
				return false
			}
			parts++
			start = index + 1
		} else if text[index] < '0' || text[index] > '9' {
			return false
		}
	}
	return parts == 4
}

// Warm starts the worker in the background so Connect does not wait for a process start.
func (s *Service) Warm() *pyjson.Object {
	s.mu.Lock()
	idle := s.process == nil
	s.mu.Unlock()
	if _, err := os.Stat(s.executable); idle && err == nil {
		go func() {
			if _, err := s.worker(); err != nil {
				s.setMessage(err.Error())
			}
		}()
	}
	return s.Status()
}

func (s *Service) setMessage(text string) {
	s.mu.Lock()
	s.message = text
	s.mu.Unlock()
	s.changed()
}

func (s *Service) worker() (worker.Process, error) {
	s.spawnMu.Lock()
	defer s.spawnMu.Unlock()
	s.mu.Lock()
	current := s.process
	s.mu.Unlock()
	if current != nil && current.Alive() {
		return current, nil
	}
	process, err := s.spawn(s.executable, s.event)
	if err != nil {
		return nil, err
	}
	hello, err := process.Request("hello", worker.DefaultTimeout, nil)
	if err != nil {
		return nil, err // Python leaves this worker running; it ends with the backend
	}
	if !control.HasCapability(hello, "explore") {
		process.Close()
		return nil, apperr.RuntimeError("The live client is outdated; rebuild the native programs")
	}
	s.mu.Lock()
	s.process = process
	s.mu.Unlock()
	return process, nil
}

func (s *Service) alive() bool {
	s.mu.Lock()
	process := s.process
	s.mu.Unlock()
	return process != nil && process.Alive()
}

func (s *Service) event(event *pyjson.Object) {
	kind, _ := event.Get("event")
	state, _ := event.Get("state")
	if kind != "live_connection" || state != "closed" {
		return
	}
	defer s.changed()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == "connected" {
		s.state, s.message = "lost", "The device closed the connection. Press Connect to try again."
		closeOnce(s.pollStop)
	}
}

// request is _request: refused unless connected; timeout in seconds as in Python.
func (s *Service) request(op string, timeout time.Duration, fields *pyjson.Object) (any, error) {
	s.mu.Lock()
	process, state := s.process, s.state
	s.mu.Unlock()
	if process == nil || state != "connected" {
		return nil, apperr.ValueError("Connect to a device first")
	}
	return process.Request(op, timeout, fields)
}

// Endpoint returns the connected (or last) endpoint; ok is false when there is none.
func (s *Service) Endpoint() (string, any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.host, s.port, s.port != nil
}

// Status is the page's view: state, endpoint (or the remembered one), watch with values.
func (s *Service) Status() *pyjson.Object {
	var remembered *pyjson.Object
	if s.memory != nil {
		section, _ := s.memory.Get().Get("explorer")
		remembered = section.(*pyjson.Object)
	} else {
		remembered = pyjson.NewObject().Set("host", "").Set("port", int64(102))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	host, port := any(s.host), s.port
	if s.port == nil {
		host, _ = remembered.Get("host")
		port, _ = remembered.Get("port")
	}
	watch := make([]any, 0, len(s.watch))
	for _, item := range s.watch {
		merged := copyObject(item)
		value, found := s.values[pytext.Str(get(item, "ref"))]
		if !found {
			value = pyjson.NewObject().Set("text", "").Set("status", "pending").Set("at", nil)
		}
		for _, key := range value.Keys() {
			merged.Set(key, get(value, key))
		}
		watch = append(watch, merged)
	}
	return pyjson.NewObject().Set("state", s.state).Set("message", s.message).Set("host", host).Set("port", port).
		Set("watch", watch).Set("interval_ms", s.interval).Set("polling", s.polling).Set("reads", s.reads)
}

// Activity is the explorer part of /activity.
func (s *Service) Activity() *pyjson.Object {
	s.mu.Lock()
	defer s.mu.Unlock()
	host := ""
	if s.port != nil {
		host = s.host
	}
	return pyjson.NewObject().Set("state", s.state).Set("host", host).Set("polling", s.polling).
		Set("watched", int64(len(s.watch)))
}

func copyObject(source *pyjson.Object) *pyjson.Object {
	result := pyjson.NewObject()
	for _, key := range source.Keys() {
		result.Set(key, get(source, key))
	}
	return result
}

func closeOnce(stop chan struct{}) {
	defer func() { recover() }()
	close(stop)
}

// isTimeout reports a worker deadline (Python's TimeoutError, not a RuntimeError).
func isTimeout(err error) bool {
	var timeout *worker.Timeout
	return errors.As(err, &timeout)
}
