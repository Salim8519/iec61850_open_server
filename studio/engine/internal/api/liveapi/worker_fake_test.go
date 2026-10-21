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
