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

// OpenRunLog starts a run log under root (python: RunLog(directory / "logs")).
func OpenRunLog(root string) (*RunLog, error) {
	log := &RunLog{RunID: newHexID(), root: root, records: make(chan *pyjson.Object, 4096), done: make(chan struct{})}
	log.path = filepath.Join(root, log.RunID)
	if err := os.MkdirAll(log.path, 0o755); err != nil {
		return nil, err
	}
	started := pyval.Obj("kind", "run_started", "run_id", log.RunID, "at", pythonNow(), "application", "0.1.0")
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

func (l *RunLog) run() {
	defer close(l.done)
	var (
		file    *os.File
		writer  *bufio.Writer
		index   int
		size    int
		pending int
	)
	for event := range l.records {
		record := pyjson.NewObject().Set("run_id", l.RunID)
		for _, key := range event.Keys() {
			value, _ := event.Get(key)
			record.Set(key, value)
		}
		encoded, err := pyjson.Compact(record)
		if err != nil {
			continue
		}
		encoded = append(encoded, '\n')
		if file != nil && size+len(encoded) > segmentBytes {
			_ = writer.Flush()
			_ = file.Close()
			file = nil
			index++
			size = 0
		}
		if file == nil {
			name := fmt.Sprintf("events-%04d.jsonl", index)
			file, err = os.OpenFile(filepath.Join(l.path, name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				continue
			}
			writer = bufio.NewWriterSize(file, 64*1024)
			appendLine(filepath.Join(l.path, "manifest.jsonl"), pyval.Obj("kind", "segment_opened",
				"run_id", l.RunID, "at", pythonNow(), "file", name))
		}
		_, _ = writer.Write(encoded)
		size += len(encoded)
		pending++
		if kind, _ := event.Get("kind"); pending >= 100 || kind != "signal" {
			_ = writer.Flush()
			pending = 0
		}
	}
	if file != nil {
		_ = writer.Flush()
		_ = file.Close()
	}
	closed := pyval.Obj("kind", "run_closed", "run_id", l.RunID, "at", pythonNow())
	appendLine(filepath.Join(l.path, "manifest.jsonl"), closed)
	appendLine(filepath.Join(l.root, "runs.jsonl"), closed)
}
