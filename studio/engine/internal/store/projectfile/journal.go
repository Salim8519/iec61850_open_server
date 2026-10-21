package projectfile

import (
	"os"
	"path/filepath"
	"time"

	"studio-engine/internal/pyjson"
)

// JournalName is the append-only journal inside <data>/logs. History is never rewritten.
const JournalName = "project-file-migrations.jsonl"

// Record is one journal line: what happened to one app-owned file. From and To are relative
// to the data folder with "/" separators. Reason or Error is set for everything but a rename.
type Record struct {
	Kind, At, Correlation, From, To, Reason, Error string
}

// journal appends the lines of one startup; Correlation ties them together.
type journal struct {
	dataDir     string
	correlation string
}

func newJournal(dataDir string) *journal { return &journal{dataDir: dataDir, correlation: uuid4Hex()} }

// write appends the record (a failed append is not fatal: the rename already happened) and
// returns it. reason or failure may be empty; a non-empty one is added to the line.
func (j *journal) write(kind, from, to, reason, failure string) Record {
	record := Record{Kind: kind, At: pythonNow(), Correlation: j.correlation, From: j.relative(from),
		To: j.relative(to), Reason: reason, Error: failure}
	line := pyjson.NewObject().Set("kind", kind).Set("at", record.At).Set("correlation_id", j.correlation).
		Set("from", record.From).Set("to", record.To)
	if reason != "" {
		line.Set("reason", reason)
	}
	if failure != "" {
		line.Set("error", failure)
	}
	if encoded, err := pyjson.Compact(line); err == nil {
		j.append(append(encoded, '\n'))
	}
	return record
}

func (j *journal) append(line []byte) {
	folder := filepath.Join(j.dataDir, "logs")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(filepath.Join(folder, JournalName), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.Write(line)
}

func (j *journal) relative(path string) string {
	relative, err := filepath.Rel(j.dataDir, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(relative)
}

// pythonNow is datetime.now(timezone.utc).isoformat(): microseconds only when not zero.
func pythonNow() string {
	now := time.Now().UTC()
	if now.Nanosecond()/1000 == 0 {
		return now.Format("2006-01-02T15:04:05") + "+00:00"
	}
	return now.Format("2006-01-02T15:04:05.000000") + "+00:00"
}
