// Package audit keeps the owner page's action record: one JSON line per sign-in,
// failed try and change, appended and synced, never rewritten or deleted by the app.
package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry is one recorded owner action. Detail never holds passwords, codes or
// full license keys.
type Entry struct {
	Time   int64  `json:"time"`
	Action string `json:"action"`
	IP     string `json:"ip,omitempty"`
	Detail string `json:"detail,omitempty"`
	OK     bool   `json:"ok"`
}

// Log appends to <data>/admin-audit.jsonl.
type Log struct {
	path string
	mu   sync.Mutex
}

// Path is the audit file inside the data folder.
func Path(dataDir string) string { return filepath.Join(dataDir, "admin-audit.jsonl") }

// Open returns the log for dataDir; the file is created on the first entry.
func Open(dataDir string) *Log { return &Log{path: Path(dataDir)} }

// Add writes one entry and syncs it to disk.
func (l *Log) Add(action, ip, detail string, ok bool, now time.Time) error {
	line, err := json.Marshal(Entry{Time: now.Unix(), Action: action, IP: ip, Detail: detail, OK: ok})
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err = file.Write(append(line, '\n')); err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

// Recent returns up to limit newest entries, newest first.
func (l *Log) Recent(limit int) ([]Entry, error) {
	file, err := os.Open(l.path)
	if errors.Is(err, os.ErrNotExist) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	ring := make([]Entry, 0, limit)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 16<<10), 64<<10)
	for scanner.Scan() {
		var entry Entry
		if json.Unmarshal(scanner.Bytes(), &entry) != nil {
			continue
		}
		if len(ring) == limit {
			ring = ring[1:]
		}
		ring = append(ring, entry)
	}
	for i, j := 0, len(ring)-1; i < j; i, j = i+1, j-1 {
		ring[i], ring[j] = ring[j], ring[i]
	}
	return ring, scanner.Err()
}
