// Package numlog is gateway/traffic_log.py's NumberedLog: a bounded, numbered list of
// entries (gateway and master traffic and commands) that pages read after a sequence.
package numlog

import (
	"sync"
	"sync/atomic"
	"time"

	"studio-engine/internal/pyjson"
)

// Capacities used by the gateway and the master (traffic_log.py).
const (
	TrafficLines = 1000
	CommandLines = 200
)

// Log keeps the newest capacity entries; seq restarts at 1 per process like Python.
type Log struct {
	mu       sync.Mutex
	capacity int
	entries  []*pyjson.Object
	sequence int64
	watch    atomic.Pointer[func()]
}

// New returns an empty log.
func New(capacity int) *Log { return &Log{capacity: capacity} }

// Watch sets fn to run after every Add, Update and Clear, outside the log's lock (the push
// stream's change hook); nil removes it. fn must not block.
func (l *Log) Watch(fn func()) {
	if fn == nil {
		l.watch.Store(nil)
		return
	}
	l.watch.Store(&fn)
}

func (l *Log) notify() {
	if fn := l.watch.Load(); fn != nil {
		(*fn)()
	}
}

// Add appends entry (its keys in call order) plus "seq" and "at" (time.time()).
func (l *Log) Add(entry *pyjson.Object) *pyjson.Object {
	defer l.notify()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sequence++
	entry.Set("seq", l.sequence).Set("at", Now())
	l.entries = append(l.entries, entry)
	if len(l.entries) > l.capacity {
		l.entries = append(l.entries[:0:0], l.entries[len(l.entries)-l.capacity:]...)
	}
	return copyOf(entry)
}

// Sequence is the last seq handed out (the status' traffic_last).
func (l *Log) Sequence() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.sequence
}

// After returns {"entries": newest limit entries with seq > seq, "last", "dropped"}.
func (l *Log) After(seq int64, limit int) *pyjson.Object {
	l.mu.Lock()
	fresh := make([]any, 0)
	for _, entry := range l.entries {
		if value, _ := entry.Get("seq"); value.(int64) > seq {
			fresh = append(fresh, copyOf(entry))
		}
	}
	last := l.sequence
	l.mu.Unlock()
	dropped := len(fresh) - limit
	if dropped < 0 {
		dropped = 0
	}
	return pyjson.NewObject().Set("entries", fresh[dropped:]).Set("last", last).Set("dropped", int64(dropped))
}

// Update merges changes into the entry with seq (dict.update keeps existing key places).
func (l *Log) Update(seq int64, changes *pyjson.Object) {
	defer l.notify()
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, entry := range l.entries {
		if value, _ := entry.Get("seq"); value.(int64) == seq {
			for _, key := range changes.Keys() {
				item, _ := changes.Get(key)
				entry.Set(key, item)
			}
			return
		}
	}
}

// Clear removes every entry; the sequence keeps counting.
func (l *Log) Clear() {
	defer l.notify()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = nil
}

// Now is time.time() as a float of seconds.
func Now() float64 { return float64(time.Now().UnixNano()) / 1e9 }

func copyOf(entry *pyjson.Object) *pyjson.Object {
	result := pyjson.NewObject()
	for _, key := range entry.Keys() {
		value, _ := entry.Get(key)
		result.Set(key, value)
	}
	return result
}
