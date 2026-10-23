package runtime

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// segmentBytes is observability/log_writer.py SEGMENT_BYTES.
const segmentBytes = 10 * 1024 * 1024

// RunLog writes <logs>/<run_id>/events-NNNN.jsonl, manifest.jsonl and <logs>/runs.jsonl with
// the bytes RunLog in log_writer.py writes, on its own goroutine (0111 rule 6): callers only
// enqueue. Records are json.dumps(..., separators=(",", ":")) + "\n", flushed after 100
// signal events or any other event.
type RunLog struct {
	RunID   string
	root    string
	path    string
	records chan *pyjson.Object
	done    chan struct{}
}

func pythonNow() string {
	now := time.Now().UTC()
	if now.Nanosecond()/1000 == 0 {
		return now.Format("2006-01-02T15:04:05") + "+00:00"
	}
	return now.Format("2006-01-02T15:04:05.000000") + "+00:00"
}

func appendLine(path string, value *pyjson.Object) {
	encoded, err := pyjson.Compact(value)
	if err != nil {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.Write(append(encoded, '\n'))
}

// OpenRunLog starts a run log under root (python: RunLog(directory / "logs")); application is
// the app version recorded in run_started ("unknown" when the caller has none).
func OpenRunLog(root, application string) (*RunLog, error) {
	if application == "" {
		application = "unknown"
	}
	log := &RunLog{RunID: newHexID(), root: root, records: make(chan *pyjson.Object, 4096), done: make(chan struct{})}
	log.path = filepath.Join(root, log.RunID)
	if err := os.MkdirAll(log.path, 0o755); err != nil {
		return nil, err
	}
	started := pyval.Obj("kind", "run_started", "run_id", log.RunID, "at", pythonNow(), "application", application)
	appendLine(filepath.Join(log.path, "manifest.jsonl"), started)
	appendLine(filepath.Join(root, "runs.jsonl"), started)
	go log.run()
	return log, nil
}

// Write enqueues one bus event (the envelope with id, kind and timestamp).
func (l *RunLog) Write(event *pyjson.Object) { l.records <- event }

// Close drains the queue, closes the segment and appends run_closed.
func (l *RunLog) Close() {
	close(l.records)
	<-l.done
}

// segment is the open events-NNNN.jsonl of a run log.
type segment struct {
	log     *RunLog
	file    *os.File
	writer  *bufio.Writer
	index   int
	size    int
	pending int
}

// write appends one record (the run_id first, then the event's keys) and flushes after 100
// signal records or any other record.
func (s *segment) write(event *pyjson.Object) {
	record := pyjson.NewObject().Set("run_id", s.log.RunID)
	for _, key := range event.Keys() {
		value, _ := event.Get(key)
		record.Set(key, value)
	}
	encoded, err := pyjson.Compact(record)
	if err != nil {
		return
	}
	encoded = append(encoded, '\n')
	if s.file != nil && s.size+len(encoded) > segmentBytes {
		_ = s.writer.Flush()
		_ = s.file.Close()
		s.file = nil
		s.index++
		s.size = 0
	}
	if s.file == nil {
		name := fmt.Sprintf("events-%04d.jsonl", s.index)
		s.file, err = os.OpenFile(filepath.Join(s.log.path, name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			s.file = nil
			return
		}
		s.writer = bufio.NewWriterSize(s.file, 64*1024)
		appendLine(filepath.Join(s.log.path, "manifest.jsonl"), pyval.Obj("kind", "segment_opened",
			"run_id", s.log.RunID, "at", pythonNow(), "file", name))
	}
	_, _ = s.writer.Write(encoded)
	s.size += len(encoded)
	s.pending++
	if kind, _ := event.Get("kind"); s.pending >= 100 || kind != "signal" {
		_ = s.writer.Flush()
		s.pending = 0
	}
}

func (s *segment) close() {
	if s.file != nil {
		_ = s.writer.Flush()
		_ = s.file.Close()
	}
}

func (l *RunLog) run() {
	defer close(l.done)
	out := &segment{log: l}
	open := newSummaries()
	flush := func(before int64, all bool) {
		for _, record := range open.due(before, all) {
			out.write(record)
		}
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case event, ok := <-l.records:
			if !ok {
				flush(0, true)
				out.close()
				closed := pyval.Obj("kind", "run_closed", "run_id", l.RunID, "at", pythonNow())
				appendLine(filepath.Join(l.path, "manifest.jsonl"), closed)
				appendLine(filepath.Join(l.root, "runs.jsonl"), closed)
				return
			}
			second, timed := eventSecond(event)
			if timed {
				flush(second, false)
			}
			if timed && summarised(event) {
				open.add(event, second)
				continue
			}
			out.write(event)
		case now := <-ticker.C:
			flush(now.Unix()-1, false) // a window closes once its second is over (and a margin)
		}
	}
}
