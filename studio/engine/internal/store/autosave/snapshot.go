package autosave

import (
	"os"
	"path/filepath"
	"sync"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/archive"
)

// SnapshotWriter writes full autosave snapshots on one background goroutine. Requests
// coalesce: while a snapshot is being written only the latest requested document is kept,
// so a burst of mutations costs one extra write. After a snapshot is renamed in, the
// journal (when given) keeps only the records newer than it.
type SnapshotWriter struct {
	dataDir string
	enc     *archive.Encoder
	journal *Journal
	onError func(rev uint64, err error)

	mu        sync.Mutex
	changed   *sync.Cond
	pending   *snapshotRequest
	requested uint64 // highest revision requested
	finished  uint64 // highest revision attempted
	lastErr   error  // result of the attempt that set finished
	closed    bool
	stopped   chan struct{}
}

type snapshotRequest struct {
	rev uint64
	doc *pyjson.Object
}

// NewSnapshotWriter starts the writer goroutine. enc and journal may be nil; onError (may
// be nil) receives every failed snapshot, which is also returned by the next Flush.
func NewSnapshotWriter(dataDir string, enc *archive.Encoder, journal *Journal,
	onError func(rev uint64, err error)) *SnapshotWriter {
	if enc == nil {
		enc = archive.NewEncoder()
	}
	w := &SnapshotWriter{dataDir: dataDir, enc: enc, journal: journal, onError: onError,
		stopped: make(chan struct{})}
	w.changed = sync.NewCond(&w.mu)
	go w.run()
	return w
}

// Request asks for a snapshot of compactDoc (the document of commit rev; it must not be
// mutated afterwards). It never waits for disk I/O. Older revisions than an earlier
// request are ignored.
func (w *SnapshotWriter) Request(rev uint64, compactDoc *pyjson.Object) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || rev < w.requested {
		return
	}
	w.requested = rev
	w.pending = &snapshotRequest{rev: rev, doc: compactDoc}
	w.changed.Broadcast()
}

// Flush waits until the latest request made before the call was written and returns the
// error of that attempt (nil when it succeeded).
func (w *SnapshotWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	target := w.requested
	for w.finished < target {
		w.changed.Wait()
	}
	if w.finished == target {
		return w.lastErr
	}
	return nil
}

// Close flushes, stops the goroutine and closes the journal (removing it when the last
// snapshot holds every record). Requests after Close are ignored.
func (w *SnapshotWriter) Close() error {
	err := w.Flush()
	w.mu.Lock()
	w.closed = true
	w.changed.Broadcast()
	w.mu.Unlock()
	<-w.stopped
	if w.journal != nil {
		if closeErr := w.journal.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

func (w *SnapshotWriter) run() {
	defer close(w.stopped)
	for {
		w.mu.Lock()
		for w.pending == nil && !w.closed {
			w.changed.Wait()
		}
		request := w.pending
		w.pending = nil
		w.mu.Unlock()
		if request == nil {
			return
		}
		err := w.write(request)
		w.mu.Lock()
		w.finished, w.lastErr = request.rev, err
		w.changed.Broadcast()
		w.mu.Unlock()
		if err != nil && w.onError != nil {
			w.onError(request.rev, err)
		}
	}
}

// write encodes the document, writes autosave.tmp, marks the snapshot in the journal,
// renames it onto autosave.subsim and trims the journal. No lock is held during I/O.
func (w *SnapshotWriter) write(request *snapshotRequest) error {
	payload, err := w.enc.Encode(request.doc)
	if err != nil {
		return err
	}
	temporary := filepath.Join(w.dataDir, TempName)
	if err := os.WriteFile(temporary, payload, 0o666); err != nil {
		return err
	}
	sha := digest(payload)
	if w.journal != nil {
		if err := w.journal.markSnapshot(request.rev, sha); err != nil {
			return err
		}
	}
	if err := os.Rename(temporary, filepath.Join(w.dataDir, AutosaveName)); err != nil {
		return err
	}
	if w.journal != nil {
		return w.journal.snapshotted(request.rev, sha)
	}
	return nil
}
