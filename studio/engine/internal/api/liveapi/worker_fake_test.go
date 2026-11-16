package liveapi

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"studio-engine/internal/acquisition"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// scriptedWorkers replays the native conversation Python's fake workers recorded:
// a request is answered with the reply recorded for the same process role, op and args.
type scriptedWorkers struct {
	mu      sync.Mutex
	replies map[string]any
	workers []*scriptedWorker
	misses  []string
	reads   int
}

type scriptedWorker struct {
	owner   *scriptedWorkers
	role    string
	onEvent func(*pyjson.Object)
	ops     map[string]bool
}

func newScript(entries []any) *scriptedWorkers {
	script := &scriptedWorkers{replies: map[string]any{}}
	for _, entry := range entries {
		items := pyv.List(entry)
		script.replies[pyv.Str(items[0])+"|"+pyv.Str(items[1])+"|"+pyv.Str(items[2])] = items[3]
	}
	return script
}

func canonical(args *pyjson.Object) string {
	body, err := pyjson.Flask(args)
	if err != nil {
		panic(err)
	}
	return strings.TrimSuffix(string(body), "\n")
}

func (s *scriptedWorkers) start(executable string, onEvent func(*pyjson.Object)) (acquisition.Worker, error) {
	role := "live"
	if filepath.Base(executable) == "studio_goose.exe" {
		role = "goose"
	}
	worker := &scriptedWorker{owner: s, role: role, onEvent: onEvent, ops: map[string]bool{}}
	s.mu.Lock()
	s.workers = append(s.workers, worker)
	s.mu.Unlock()
	return worker, nil
}

func (w *scriptedWorker) Request(op string, _ time.Duration, args *pyjson.Object) (any, error) {
	key := w.role + "|" + op + "|" + canonical(args)
	if op == "read" && controlCheckRead(args) {
		// The device control check (ADR-0138) is a Go-only read Python never recorded. It fails
		// here, as on a device that cannot answer, so the recorded conversation stays exact.
		return nil, apperr.RuntimeError("Go-only control check: no recorded reply")
	}
	s := w.owner
	s.mu.Lock()
	defer s.mu.Unlock()
	w.ops[op] = true
	if op == "read" {
		s.reads++
	}
	reply, found := s.replies[key]
	if !found {
		s.misses = append(s.misses, key)
		return nil, apperr.RuntimeError(fmt.Sprintf("no recorded reply for %s", key))
	}
	return pyv.Copy(reply), nil
}

// controlCheckRead is a read of ctlModel (and sboTimeout, Loc.stVal) as the control check asks.
func controlCheckRead(args *pyjson.Object) bool {
	items, models := pyv.Objects(pyv.GetList(args, "refs")), 0
	for _, item := range items {
		ref := pyv.GetStr(item, "ref")
		switch {
		case strings.HasSuffix(ref, ".ctlModel") && pyv.GetStr(item, "fc") == "CF":
			models++
		case strings.HasSuffix(ref, ".sboTimeout") && pyv.GetStr(item, "fc") == "CF":
		case strings.HasSuffix(ref, ".Loc.stVal") && pyv.GetStr(item, "fc") == "ST":
		default:
			return false
		}
	}
	return models > 0
}

func (w *scriptedWorker) Close() {}

// deliver sends an event to the worker of role that handled op (e.g. the GOOSE listener).
func (s *scriptedWorkers) deliver(role, op string, event *pyjson.Object) bool {
	s.mu.Lock()
	var target *scriptedWorker
	for _, worker := range s.workers {
		if worker.role == role && (op == "" || worker.ops[op]) {
			target = worker
		}
	}
	s.mu.Unlock()
	if target == nil {
		return false
	}
	target.onEvent(event)
	return true
}

func (s *scriptedWorkers) readCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reads
}
