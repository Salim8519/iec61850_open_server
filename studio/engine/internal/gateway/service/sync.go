package service

import (
	"sort"
	"strings"
	"sync"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/worker"
)

// Timing of the value push. Python diffed every point every 0.1 s; here a point is
// recomputed only after the hub reports a change of its value, quality, timestamp, IED or
// run state, and batches still leave at most every 100 ms (the same coalescing, so SCADA
// sees the same spontaneous events). FullSweep re-checks every point as a safety net for
// state changes the hub might not report (project edits of an IED's role).
const (
	MinGap    = 100 * time.Millisecond
	FullSweep = time.Second
	batchSize = 500
)

type monitored struct {
	id, ied, ref string
	remote       bool
	raw          *pyjson.Object
}

type syncer struct {
	manager *Manager
	process worker.Process
	points  []monitored
	sent    map[string]Sample
	byIED   map[string][]int
	remote  []int

	mu    sync.Mutex
	deps  map[[2]string][]int // (ied, ref) of a value, its quality and its timestamp -> points
	dirty map[int]bool
	all   bool

	wake    chan struct{}
	stop    chan struct{}
	done    chan struct{}
	cancel  func()
	closing sync.Once
	sweep   time.Duration
}

func newSyncer(manager *Manager, process worker.Process, points []*pyjson.Object, sweep time.Duration) *syncer {
	s := &syncer{manager: manager, process: process, sent: map[string]Sample{}, byIED: map[string][]int{},
		deps: map[[2]string][]int{}, dirty: map[int]bool{}, all: true, wake: make(chan struct{}, 1),
		stop: make(chan struct{}), done: make(chan struct{}), sweep: sweep}
	for _, point := range points {
		if kind, _ := get(point, "kind").(string); strings.HasPrefix(kind, "command_") {
			continue
		}
		item := monitored{id: text(point, "id"), ied: text(point, "ied"), ref: text(point, "ref"),
			remote: get(point, "source") == "remote", raw: point}
		index := len(s.points)
		s.points = append(s.points, item)
		if item.remote {
			s.remote = append(s.remote, index)
		} else {
			s.byIED[item.ied] = append(s.byIED[item.ied], index)
		}
	}
	return s
}

func (s *syncer) start() {
	s.cancel = s.manager.runtime.Subscribe(s.notify)
	s.signal() // the first batch (every point) leaves after 100 ms, as Python's first tick
	go s.run()
}

// notify marks the points a change touches; it never blocks the hub.
func (s *syncer) notify(change Change) {
	s.mu.Lock()
	marked := false
	switch {
	case change.IEDID == "":
		s.all, marked = true, true
	case change.Ref == "":
		for _, index := range s.byIED[change.IEDID] {
			s.dirty[index], marked = true, true
		}
	default:
		for _, index := range s.deps[[2]string{change.IEDID, change.Ref}] {
			s.dirty[index], marked = true, true
		}
	}
	s.mu.Unlock()
	if marked {
		s.signal()
	}
}

// MarkRemote marks the converter's points after the master merged new device values.
func (s *syncer) markRemote() {
	if len(s.remote) == 0 {
		return
	}
	s.mu.Lock()
	for _, index := range s.remote {
		s.dirty[index] = true
	}
	s.mu.Unlock()
	s.signal()
}

func (s *syncer) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *syncer) run() {
	defer close(s.done)
	var sweep <-chan time.Time
	if s.sweep > 0 {
		ticker := time.NewTicker(s.sweep)
		defer ticker.Stop()
		sweep = ticker.C
	}
	last := time.Now()
	for {
		select {
		case <-s.stop:
			return
		case <-s.wake:
		case <-sweep:
			s.mu.Lock()
			s.all = true
			s.mu.Unlock()
		}
		if wait := time.Until(last.Add(MinGap)); wait > 0 {
			select {
			case <-s.stop:
				return
			case <-time.After(wait):
			}
		}
		last = time.Now()
		if _, err := s.flush(); err != nil {
			s.manager.setMessage("Values could not be sent: " + err.Error())
		}
	}
}

// flush sends the changed samples of the marked points, in point order, 500 per request.
func (s *syncer) flush() (int, error) {
	s.mu.Lock()
	all, dirty := s.all, s.dirty
	s.all, s.dirty = false, map[int]bool{}
	s.mu.Unlock()
	var indexes []int
	if all {
		indexes = make([]int, len(s.points))
		for index := range indexes {
			indexes[index] = index
		}
		s.rebuildDeps()
	} else {
		for index := range dirty {
			indexes = append(indexes, index)
		}
		sort.Ints(indexes)
	}
	if len(indexes) == 0 {
		return 0, nil
	}
	runState := s.manager.runtime.RunState()
	ieds := s.manager.iedIndex()
	var changed []any
	for _, index := range indexes {
		point := s.points[index]
		var current Sample
		if point.remote && s.manager.Converter != nil {
			var err error
			if current, err = s.manager.Converter.Sample(point.raw); err != nil {
				return 0, err
			}
		} else {
			current = SampleOf(point.id, point.ied, point.ref, ieds[point.ied], runState, s.manager.runtime)
		}
		if previous, found := s.sent[point.id]; !found || !previous.Same(current) {
			s.sent[point.id] = current
			changed = append(changed, current.Object())
		}
	}
	for start := 0; start < len(changed); start += batchSize {
		end := min(start+batchSize, len(changed))
		if _, err := s.process.Request("update", worker.DefaultTimeout,
			pyjson.NewObject().Set("values", changed[start:end])); err != nil {
			return 0, err
		}
	}
	return len(changed), nil
}

// rebuildDeps maps each value, quality and timestamp reference to the points that carry it.
func (s *syncer) rebuildDeps() {
	deps := map[[2]string][]int{}
	for index, point := range s.points {
		if point.remote {
			continue
		}
		deps[[2]string{point.ied, point.ref}] = append(deps[[2]string{point.ied, point.ref}], index)
		if signal, found := s.manager.runtime.Signal(point.ied, point.ref); found {
			for _, ref := range []string{signal.QualityRef, signal.TimestampRef} {
				if ref != "" && ref != point.ref {
					deps[[2]string{point.ied, ref}] = append(deps[[2]string{point.ied, ref}], index)
				}
			}
		}
	}
	s.mu.Lock()
	s.deps = deps
	s.mu.Unlock()
}

// halt ends the loop without waiting (worker_failed arrives on the worker's reader).
func (s *syncer) halt() {
	s.closing.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
		close(s.stop)
	})
}

// close ends the loop and waits up to 5 s for it (Python joins the sync thread 5 s).
func (s *syncer) close() {
	s.halt()
	select {
	case <-s.done:
	case <-time.After(5 * time.Second):
	}
}
