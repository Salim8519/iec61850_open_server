package acquisition

import (
	"sync"
	"time"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// ModeRefresh is how often a live session reads Mod/Beh/Sim again (real_session.py:10).
const ModeRefresh = 10 * time.Second

var liveCapabilities = []string{"live_discover", "live_reports", "live_poll", "read_only_client"}

// RealSession is one read-only connection to a real IED: connect, compare, report or poll,
// recover (acquisition/real_session.py). One goroutine owns the studio_live process; the
// status is guarded by mu and published as copies, never shared mutable.
type RealSession struct {
	deps    Deps
	values  *LiveValues
	ied     *pyjson.Object
	watched []any

	mu       sync.Mutex
	status   *pyjson.Object
	process  Worker
	stopping chan struct{}
	stopOnce sync.Once
	lost     chan struct{}
	lostOnce *sync.Once
	done     chan struct{}
}

// NewRealSession prepares a session; Start runs it.
func NewRealSession(deps Deps, values *LiveValues, ied *pyjson.Object, watched []any) *RealSession {
	host, port, origin := RealEndpoint(ied)
	return &RealSession{deps: deps, values: values, ied: ied, watched: watched,
		stopping: make(chan struct{}), done: make(chan struct{}),
		status: pyv.O("ied_id", pyv.Get(ied, "id"), "name", pyv.Get(ied, "name"), "state", "starting",
			"host", host, "port", port, "endpoint_source", origin, "message", "", "access", nil,
			"comparison", nil, "polling", []any{}, "health", nil)}
}

// RealEndpoint is domain/network_settings.real_endpoint: (host, port, "override"|"scd"|"missing").
func RealEndpoint(ied *pyjson.Object) (string, any, string) {
	live := pyv.GetObj(ied, "live")
	if !pyv.Truthy(live) {
		live = pyv.O("ip", "", "port", int64(102), "access", "auto_reports", "poll_ms", int64(1000))
	}
	scd := pyv.GetStr(pyv.OrObj(ied, "scl_address"), "ip")
	override := pyv.GetStr(live, "ip")
	switch {
	case override != "":
		return override, pyv.Get(live, "port"), "override"
	case scd != "":
		return scd, pyv.Get(live, "port"), "scd"
	}
	return "", pyv.Get(live, "port"), "missing"
}

// ID is the IED id.
func (s *RealSession) ID() string { return pyv.GetStr(s.ied, "id") }

func (s *RealSession) name() string { return pyv.GetStr(s.ied, "name") }

// Status returns a copy of the session status (the live view adds comparison/health None).
func (s *RealSession) Status() *pyjson.Object {
	s.mu.Lock()
	defer s.mu.Unlock()
	return pyv.Shallow(s.status)
}

func (s *RealSession) get(key string) any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return pyv.Get(s.status, key)
}

func (s *RealSession) put(pairs ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := 0; index+1 < len(pairs); index += 2 {
		s.status.Set(pairs[index].(string), pairs[index+1])
	}
}

func (s *RealSession) set(state, message string) {
	s.put("state", state, "message", message)
	ideState := "real_" + state
	if state == "stopped" {
		ideState = "stopped"
	}
	s.deps.States.SetState(s.ID(), ideState)
	s.deps.Events.Emit("live_status", pyv.O("ied_id", pyv.Get(s.ied, "id"), "state", state, "message", message))
}

func (s *RealSession) isStopping() bool {
	select {
	case <-s.stopping:
		return true
	default:
		return false
	}
}

// Start runs the session goroutine.
func (s *RealSession) Start() {
	s.mu.Lock()
	s.resetLostLocked()
	s.mu.Unlock()
	go s.run()
}

func (s *RealSession) resetLostLocked() {
	s.lost = make(chan struct{})
	s.lostOnce = &sync.Once{}
}

func (s *RealSession) markLost() {
	s.mu.Lock()
	lost, once := s.lost, s.lostOnce
	s.mu.Unlock()
	once.Do(func() { close(lost) })
}

func (s *RealSession) onEvent(event *pyjson.Object) {
	switch pyv.GetStr(event, "event") {
	case "live_report":
		values := pyv.GetObj(event, "values")
		if !pyv.Has(event, "values") {
			values = pyjson.NewObject()
		}
		_, _ = s.values.Apply(s.ID(), values, "live_report")
	case "live_connection":
		if pyv.Equal(pyv.Get(event, "state"), "closed") && !s.isStopping() {
			s.markLost()
		}
	case "worker_failed":
		if !s.isStopping() {
			s.markLost()
		}
	}
}

// Read reads items now ("Read these now" or polling) and applies them as live values.
func (s *RealSession) Read(items []any, source string) (*pyjson.Object, error) {
	s.mu.Lock()
	process, state := s.process, pyv.GetStr(s.status, "state")
	s.mu.Unlock()
	if process == nil || state != "live" {
		return nil, apperr.ValueError(s.name() + " is not connected yet")
	}
	if len(items) > 2000 {
		items = items[:2000]
	}
	answer, err := process.Request("read", 30*time.Second, pyv.O("refs", items))
	if err != nil {
		return nil, err
	}
	values := pyjson.NewObject()
	for _, entry := range pyv.Objects(pyv.GetList(pyv.Obj(answer), "values")) {
		switch {
		case pyv.Has(entry, "value"):
			values.Set(pyv.GetStr(entry, "ref"), pyv.Get(entry, "value"))
		case pyv.Has(entry, "structure"):
			pyv.Update(values, pyv.OrObj(entry, "structure"))
		default:
			s.deps.Events.Emit("live_read_error", pyv.O("ied_id", pyv.Get(s.ied, "id"), "ref", pyv.Get(entry, "ref"),
				"message", "The device could not read this point (error "+pyv.PyStr(pyv.Get(entry, "error"))+")"))
		}
	}
	return s.values.Apply(s.ID(), values, source)
}

// RefreshHealth reads report-block, GOOSE-block and mode settings again on this association.
func (s *RealSession) RefreshHealth() (*pyjson.Object, error) {
	s.mu.Lock()
	process, state := s.process, pyv.GetStr(s.status, "state")
	s.mu.Unlock()
	if process == nil || state != "live" {
		return nil, apperr.ValueError(s.name() + " is not connected yet")
	}
	access := pyv.GetStr(pyv.GetObj(s.ied, "live"), "access")
	blocks, err := s.readBlocks(process, access)
	if err != nil {
		return nil, err
	}
	health, err := readHealth(process, s.ied, blocks, access, s.deps.Clock.Now)
	if err != nil {
		return nil, err
	}
	s.put("health", health)
	return health, nil
}

// Stop disconnects (giving back every report block) and waits up to 20 s for the goroutine.
func (s *RealSession) Stop() {
	s.stopOnce.Do(func() { close(s.stopping) })
	s.closeProcess()
	select {
	case <-s.done:
	case <-time.After(20 * time.Second):
	}
	s.set("stopped", "Disconnected; every report block ScadaBench used was given back")
}

func (s *RealSession) closeProcess() {
	s.mu.Lock()
	process := s.process
	s.process = nil
	s.mu.Unlock()
	if process != nil {
		_, _ = process.Request("disconnect", 15*time.Second, pyjson.NewObject())
		process.Close()
	}
}
