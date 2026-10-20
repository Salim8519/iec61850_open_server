package desktop

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger is the launcher log of windowed.configure_logging. A nil *Logger discards.
//
// With a terminal it writes Python's basicConfig default to stderr:
//
//	WARNING:studio_app.launcher.app_window:App window was handed to another browser process
//
// Headless it appends to <data>/logs/launcher-YYYYMMDD.log (local date at start, UTF-8):
//
//	2026-10-20 15:04:05,123 WARNING studio_app.launcher.app_window: App window was ...
//
// Lines end like Python text files ("\r\n" on Windows). Python tracebacks are not
// reproduced; Error callers put the error text into the message instead.
type Logger struct {
	mu    sync.Mutex
	out   io.Writer
	file  *os.File
	timed bool
	now   func() time.Time
}

// OpenLog is windowed.configure_logging for dataDir. When the log folder or file cannot
// be created the logger discards everything (Python's NullHandler fallback).
func OpenLog(dataDir string) *Logger {
	if !Headless() {
		return &Logger{out: os.Stderr, now: time.Now}
	}
	return openLogFile(dataDir, time.Now)
}

func openLogFile(dataDir string, now func() time.Time) *Logger {
	folder := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return &Logger{out: io.Discard, now: now}
	}
	name := filepath.Join(folder, "launcher-"+now().Format("20060102")+".log")
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return &Logger{out: io.Discard, now: now}
	}
	return &Logger{out: file, file: file, timed: true, now: now}
}

// Info logs at INFO for the named logger.
func (l *Logger) Info(name, format string, args ...any) { l.write("INFO", name, format, args) }

// Warning logs at WARNING for the named logger.
func (l *Logger) Warning(name, format string, args ...any) { l.write("WARNING", name, format, args) }

// Error logs at ERROR for the named logger (logging.exception without the traceback).
func (l *Logger) Error(name, format string, args ...any) { l.write("ERROR", name, format, args) }

// Close closes the log file, if any. Later records are discarded.
func (l *Logger) Close() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
	l.out = io.Discard
}

func (l *Logger) write(level, name, format string, args []any) {
	if l == nil {
		return
	}
	message := fmt.Sprintf(format, args...)
	l.mu.Lock()
	defer l.mu.Unlock()
	var line string
	if l.timed {
		stamp := l.now()
		line = fmt.Sprintf("%s,%03d %s %s: %s", stamp.Format("2006-01-02 15:04:05"),
			stamp.Nanosecond()/int(time.Millisecond), level, name, message)
	} else {
		line = level + ":" + name + ":" + message
	}
	_, _ = io.WriteString(l.out, line+lineEnd)
}
