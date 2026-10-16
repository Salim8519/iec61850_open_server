// Package logging writes size-capped rotating JSONL logs so the disk never fills.
package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Capped rotates name -> name.1 -> name.2 ... and deletes the oldest file, so
// total usage never exceeds maxBytes * maxFiles.
type Capped struct {
	mu       sync.Mutex
	dir      string
	name     string
	maxBytes int64
	maxFiles int
	file     *os.File
	size     int64
}

// Open creates the log directory and opens (or appends to) the current file.
func Open(dir, name string, maxMegabytes, maxFiles int) (*Capped, error) {
	if maxMegabytes < 1 {
		maxMegabytes = 1
	}
	if maxFiles < 1 {
		maxFiles = 1
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	c := &Capped{dir: dir, name: name, maxBytes: int64(maxMegabytes) << 20, maxFiles: maxFiles}
	return c, c.open()
}

// NewForTest allows byte-sized limits in tests.
func NewForTest(dir string, maxBytes int64, maxFiles int) (*Capped, error) {
	c := &Capped{dir: dir, name: "server.log", maxBytes: maxBytes, maxFiles: maxFiles}
	return c, c.open()
}

func (c *Capped) path(index int) string {
	if index == 0 {
		return filepath.Join(c.dir, c.name)
	}
	return filepath.Join(c.dir, fmt.Sprintf("%s.%d", c.name, index))
}

func (c *Capped) open() error {
	file, err := os.OpenFile(c.path(0), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	c.file, c.size = file, info.Size()
	return nil
}

func (c *Capped) rotate() error {
	c.file.Close()
	os.Remove(c.path(c.maxFiles - 1))
	for index := c.maxFiles - 2; index >= 0; index-- {
		os.Rename(c.path(index), c.path(index+1))
	}
	if c.maxFiles == 1 {
		os.Remove(c.path(0))
	}
	return c.open()
}

// Event writes one JSON line with a UTC timestamp. Callers must not pass secrets.
func (c *Capped) Event(kind string, fields map[string]any) {
	record := map[string]any{"ts": time.Now().UTC().Format(time.RFC3339), "kind": kind}
	for key, value := range fields {
		record[key] = value
	}
	line, err := json.Marshal(record)
	if err != nil {
		return
	}
	line = append(line, '\n')
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.file == nil {
		return
	}
	if c.size+int64(len(line)) > c.maxBytes && c.size > 0 {
		if err := c.rotate(); err != nil {
			return
		}
	}
	n, _ := c.file.Write(line)
	c.size += int64(n)
}

// Close flushes and closes the current file.
func (c *Capped) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.file == nil {
		return nil
	}
	err := c.file.Close()
	c.file = nil
	return err
}

// MaskKey keeps only the last group of a license key for log lines.
func MaskKey(key string) string {
	if len(key) < 5 {
		return "****"
	}
	return "SUBS-****-****-****-" + key[len(key)-4:]
}
