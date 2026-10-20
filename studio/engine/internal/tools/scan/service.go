package scan

import (
	"math"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/memory"
	"studio-engine/internal/tools/numlog"
	"studio-engine/internal/tools/pytext"
	"studio-engine/internal/tools/worker"
)

// Probing limits of tools/ied_scan.py.
const (
	Parallel       = 64
	ConnectTimeout = 500 * time.Millisecond
	identifyWait   = 10 * time.Second
)

// Prober is port_open: milliseconds of a plain TCP handshake, or ok=false.
type Prober func(host string, port int64) (float64, bool)

// PortOpen opens and closes a TCP connection without sending data.
func PortOpen(host string, port int64) (float64, bool) {
	started := time.Now()
	connection, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.FormatInt(port, 10)), ConnectTimeout)
	if err != nil {
		return 0, false
	}
	connection.Close()
	return round1(float64(time.Since(started).Nanoseconds()) / 1e6), true
}

func round1(value float64) float64 { return math.RoundToEven(value*10) / 10 }

// Service is ScanService: one scan at a time, its state polled by the page every 300 ms.
type Service struct {
	executable string
	memory     *memory.Store
	probe      Prober
	spawn      worker.Spawner
	mu         sync.Mutex
	state      *pyjson.Object
	found      []*pyjson.Object
	stop       chan struct{}
	done       chan struct{}
	changes    // OnChange: the push stream's "scan" topic
}

// NewService returns an idle scanner for studio_live at executable.
func NewService(executable string, store *memory.Store, probe Prober, spawn worker.Spawner) *Service {
	if probe == nil {
		probe = PortOpen
	}
	if spawn == nil {
		spawn = worker.Spawn
	}
	s := &Service{executable: executable, memory: store, probe: probe, spawn: spawn, stop: make(chan struct{})}
	s.state = newState("idle", "", int64(102), 0, "", nil)
	return s
}

func newState(state, text string, port any, total int, message string, started any) *pyjson.Object {
	return pyjson.NewObject().Set("state", state).Set("range", text).Set("port", port).Set("total", int64(total)).
		Set("probed", int64(0)).Set("found", []any{}).Set("message", message).Set("started", started).
		Set("seconds", int64(0))
}

// CheckPort is _check_port: isinstance(port, int) (True counts) and 1..65535.
func CheckPort(port any) error {
	if number, ok := pytext.Int64(port); !ok || !pytext.IsIntOrBool(port) || number < 1 || number > 65535 {
		return apperr.ValueError("The port must be a number between 1 and 65535 (IEC 61850 uses 102)")
	}
	return nil
}

// Preview is {"range", "count", "first", "last", "port"} without probing anything.
func (s *Service) Preview(text, port any) (*pyjson.Object, error) {
	plan, err := PlanRange(text, "")
	if err != nil {
		return nil, err
	}
	if err := CheckPort(port); err != nil {
		return nil, err
	}
	return pyjson.NewObject().Set("range", plan.Range).Set("count", int64(plan.Count)).Set("first", plan.First).
		Set("last", plan.Last).Set("port", port), nil
}

// Start validates, remembers the range and port, and starts probing in the background.
func (s *Service) Start(text, port any) (*pyjson.Object, error) {
	defer s.changed()
	plan, err := PlanRange(text, "")
	if err != nil {
		return nil, err
	}
	if err := CheckPort(port); err != nil {
		return nil, err
	}
	if _, err := os.Stat(s.executable); err != nil {
		return nil, apperr.RuntimeError("The live client (studio_live.exe) is not built yet; rebuild the native programs")
	}
	s.mu.Lock()
	if s.running() {
		s.mu.Unlock()
		return nil, apperr.ValueError("A scan is already running; stop it first")
	}
	stop, done := make(chan struct{}), make(chan struct{})
	s.stop, s.done = stop, done
	s.mu.Unlock()
	if s.memory != nil {
		if _, err := s.memory.Update("find", pyjson.NewObject().Set("range", plan.Range).Set("port", port)); err != nil {
			return nil, err
		}
	}
	number, _ := pytext.Int64(port)
	s.mu.Lock()
	s.state = newState("scanning", plan.Range, port, plan.Count,
		"Looking for IEDs on "+strconv.Itoa(plan.Count)+" address(es)", numlog.Now())
	s.found = nil
	s.mu.Unlock()
	go s.run(plan.Hosts, number, port, stop, done)
	return s.Status(), nil
}

func (s *Service) running() bool {
	if s.done == nil {
		return false
	}
	select {
	case <-s.done:
		return false
	default:
		return true
	}
}

// Stop asks the scan to end and waits up to 15 s for it.
func (s *Service) Stop() *pyjson.Object {
	defer s.changed()
	s.mu.Lock()
	stop, done := s.stop, s.done
	s.mu.Unlock()
	closeOnce(stop)
	if done != nil {
		select {
		case <-done:
		case <-time.After(15 * time.Second):
		}
	}
	return s.Status()
}

func closeOnce(stop chan struct{}) {
	defer func() { recover() }()
	close(stop)
}

func stopped(stop chan struct{}) bool {
	select {
	case <-stop:
		return true
	default:
		return false
	}
}

// Status is a copy of the state; idle shows the remembered range and port.
func (s *Service) Status() *pyjson.Object {
	s.mu.Lock()
	state := copyObject(s.state)
	found := make([]any, len(s.found))
	for index, entry := range s.found {
		found[index] = copyObject(entry)
	}
	s.mu.Unlock()
	state.Set("found", found)
	if current, _ := state.Get("state"); current == "idle" && s.memory != nil {
		remembered, _ := s.memory.Get().Get("find")
		value, _ := remembered.(*pyjson.Object).Get("range")
		port, _ := remembered.(*pyjson.Object).Get("port")
		state.Set("range", value).Set("port", port)
	}
	started, _ := state.Get("started")
	if current, _ := state.Get("state"); pytext.Truthy(started) && current == "scanning" {
		state.Set("seconds", round1(numlog.Now()-started.(float64)))
	}
	return state
}

// Scanning reports a running scan (its "seconds" counter moves without other changes).
func (s *Service) Scanning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, _ := s.state.Get("state")
	return state == "scanning"
}

// Snapshot returns state, probed and total for the activity dots.
func (s *Service) Snapshot() (string, any, any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, _ := s.state.Get("state")
	probed, _ := s.state.Get("probed")
	total, _ := s.state.Get("total")
	return state.(string), probed, total
}

func copyObject(source *pyjson.Object) *pyjson.Object {
	result := pyjson.NewObject()
	for _, key := range source.Keys() {
		value, _ := source.Get(key)
		result.Set(key, value)
	}
	return result
}
