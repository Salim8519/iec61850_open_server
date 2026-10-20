package service

import (
	"fmt"
	"sync"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/snmp/worker"
	"studio-engine/internal/tools/numlog"
	toolworker "studio-engine/internal/tools/worker"
)

// Projects is the project service as the SNMP tool uses it.
type Projects interface {
	SNMP() *pyjson.Object
	SaveSNMP(section *pyjson.Object) error
}

// Change names for the push stream.
const (
	ChangeStatus   = "snmp.status"
	ChangeRequests = "snmp.requests"
	ChangeTraps    = "snmp.traps"
	ChangeValues   = "snmp.values"
)

// EngineMissing is shown when studio_snmp.exe was not built.
const EngineMissing = "The SNMP engine (studio_snmp.exe) is missing; run studio/tools/build_snmp_engine.py"

// Service is the SNMP devices tool.
type Service struct {
	projects   Projects
	executable string
	spawn      toolworker.Spawner
	logf       func(string, ...any)
	Requests   *numlog.Log
	Traps      *numlog.Log
	MIBs       *Library

	notifyMu sync.Mutex
	notify   []func(string)

	lifecycle sync.Mutex // starting and stopping the worker
	mu        sync.Mutex // the fields below; never held across a worker request
	process   toolworker.Process
	failure   string
	devices   map[string]deviceState
	reader    readerState
	listening bool
	trapText  string
}

// New returns the tool for studio_snmp at executable; dataDir holds the MIB library.
func New(projects Projects, executable, dataDir string, spawn toolworker.Spawner, logf func(string, ...any)) *Service {
	if spawn == nil {
		spawn = toolworker.Spawn
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	s := &Service{projects: projects, executable: executable, spawn: spawn, logf: logf,
		Requests: numlog.New(numlog.TrafficLines), Traps: numlog.New(numlog.TrafficLines),
		MIBs: NewLibrary(dataDir, logf), devices: map[string]deviceState{}, reader: readerState{state: "disconnected"}}
	s.Requests.Watch(func() { s.changed(ChangeRequests) })
	s.Traps.Watch(func() { s.changed(ChangeTraps); s.changed(ChangeStatus) })
	return s
}

// OnChange adds a listener for change names (the push stream).
func (s *Service) OnChange(fn func(string)) {
	s.notifyMu.Lock()
	defer s.notifyMu.Unlock()
	s.notify = append(s.notify, fn)
}

func (s *Service) changed(name string) {
	s.notifyMu.Lock()
	listeners := append([]func(string){}, s.notify...)
	s.notifyMu.Unlock()
	for _, fn := range listeners {
		fn(name)
	}
}

// Config is the project's SNMP section with defaults filled in.
func (s *Service) Config() Config { return Load(s.projects.SNMP()) }

func (s *Service) save(config Config) error {
	section, err := config.Section()
	if err != nil {
		return err
	}
	return s.projects.SaveSNMP(section)
}

// call sends one request to the worker, starting it when needed.
func (s *Service) call(op string, fields any, timeout time.Duration) (*pyjson.Object, error) {
	process, err := s.worker()
	if err != nil {
		return nil, err
	}
	var object *pyjson.Object
	if fields != nil {
		if object, err = ToObject(fields); err != nil {
			return nil, err
		}
	}
	result, err := process.Request(op, timeout, object)
	if err != nil {
		return nil, err
	}
	answer, _ := result.(*pyjson.Object)
	if answer == nil {
		answer = pyjson.NewObject()
	}
	return answer, nil
}

// worker returns the running studio_snmp, starting it and checking its capability first.
func (s *Service) worker() (toolworker.Process, error) {
	s.lifecycle.Lock()
	defer s.lifecycle.Unlock()
	s.mu.Lock()
	process := s.process
	s.mu.Unlock()
	if process != nil && process.Alive() {
		return process, nil
	}
	process, err := s.spawn(s.executable, s.onEvent)
	if err != nil {
		return nil, apperr.RuntimeError(EngineMissing)
	}
	hello, err := process.Request("hello", toolworker.DefaultTimeout, nil)
	if err != nil {
		process.Close()
		return nil, apperr.RuntimeError(fmt.Sprintf("The SNMP engine did not start: %v", err))
	}
	capabilities, _ := hello.(*pyjson.Object)
	if !hasCapability(capabilities, worker.Capability) {
		process.Close()
		return nil, apperr.RuntimeError("The SNMP engine is outdated. Rebuild it with studio/tools/build_snmp_engine.py")
	}
	s.mu.Lock()
	s.process, s.failure = process, ""
	s.mu.Unlock()
	return process, nil
}

func hasCapability(hello *pyjson.Object, want string) bool {
	if hello == nil {
		return false
	}
	list, _ := hello.Get("capabilities")
	items, _ := list.([]any)
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// releaseIfIdle stops the worker when nothing uses it, so SNMP costs nothing while off.
func (s *Service) releaseIfIdle() {
	s.lifecycle.Lock()
	defer s.lifecycle.Unlock()
	s.mu.Lock()
	idle := len(s.devices) == 0 && s.reader.state == "disconnected" && !s.listening
	process := s.process
	if idle {
		s.process = nil
	}
	s.mu.Unlock()
	if idle && process != nil {
		process.Close()
	}
}

// Close stops the worker (app shutdown).
func (s *Service) Close() {
	s.lifecycle.Lock()
	defer s.lifecycle.Unlock()
	s.mu.Lock()
	process := s.process
	s.process = nil
	s.mu.Unlock()
	if process != nil {
		process.Close()
	}
}
