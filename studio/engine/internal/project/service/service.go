// Package service is ProjectService (projects/service.py) for the Go engine: it composes
// the SCL importer, the pure project rules and the store. Mutations are serialised by one
// mutation lock (the project actor) and publish a new immutable state by atomic pointer;
// readers never wait. Every committed change is journaled and fsynced before the call
// returns (Python saved synchronously) and a coalesced background snapshot rewrites
// autosave.subsim. Project trees are copy-on-write: published objects are never mutated.
package service

import (
	"sync"
	"sync/atomic"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
	"studio-engine/internal/store/archive"
	"studio-engine/internal/store/autosave"
)

// Config wires the service to its data folder and collaborators.
type Config struct {
	DataDir  string
	Importer *scl.Importer
	// OnChange is ProjectService.on_change(revision, source): source is nil, a tab id or
	// "gateway". The event layer turns it into project_changed.
	OnChange func(revision int64, source any)
	// OnSnapshotError receives background autosave failures (logged; the journal keeps the
	// change durable).
	OnSnapshotError func(error)
	NewID           func() string
}

type state struct {
	project  *pyjson.Object
	compact  *pyjson.Object
	notice   *pyjson.Object
	revision int64
}

// Service owns the engineering project.
type Service struct {
	config    Config
	sessionID string
	encoder   *archive.Encoder
	journal   *autosave.Journal
	snapshots *autosave.SnapshotWriter
	mutate    sync.Mutex
	seq       uint64 // journal and snapshot sequence (Python's revision does not move on every save)
	current   atomic.Pointer[state]
}

// New creates the service with a new empty project. Fold the journal of an earlier run
// (autosave.Fold) before calling New; startup recovery then restores the autosave.
func New(config Config) (*Service, error) {
	if config.NewID == nil {
		config.NewID = scl.RandomID
	}
	if config.OnChange == nil {
		config.OnChange = func(int64, any) {}
	}
	s := &Service{config: config, sessionID: config.NewID(), encoder: archive.NewEncoder()}
	journal, err := autosave.OpenJournal(config.DataDir, s.encoder)
	if err != nil {
		return nil, err
	}
	s.journal = journal
	onError := func(uint64, error) {}
	if config.OnSnapshotError != nil {
		onError = func(_ uint64, err error) { config.OnSnapshotError(err) }
	}
	s.snapshots = autosave.NewSnapshotWriter(config.DataDir, s.encoder, journal, onError)
	fresh := project.NewProject(config.NewID())
	s.current.Store(&state{project: fresh, compact: project.Compact(fresh)})
	return s, nil
}

// Close writes the last snapshot and removes the journal (process exit).
func (s *Service) Close() error { return s.snapshots.Close() }

// Revision is the in-memory revision counter.
func (s *Service) Revision() int64 { return s.current.Load().revision }

// SessionID is the per-process session id of the snapshot API.
func (s *Service) SessionID() string { return s.sessionID }

// Importer is the SCL importer (schema validator and id source) the service imports with.
func (s *Service) Importer() *scl.Importer { return s.config.Importer }

// Project returns the current project tree (immutable; do not modify).
func (s *Service) Project() *pyjson.Object { return s.current.Load().project }

// Snapshot is ProjectService.snapshot().
func (s *Service) Snapshot() *pyjson.Object {
	st := s.current.Load()
	return project.Snapshot(st.project, st.notice, s.sessionID, st.revision)
}

// Export is ProjectService.export(): the archive bytes of the current project.
func (s *Service) Export() ([]byte, error) {
	return s.encoder.Encode(s.current.Load().compact)
}

// Save is ProjectService.save(): the current project written to autosave.subsim now.
func (s *Service) Save() error {
	st := s.current.Load()
	if _, err := s.encoder.Encode(st.compact); err != nil {
		return err
	}
	s.mutate.Lock()
	s.seq++
	s.snapshots.Request(s.seq, st.compact)
	s.mutate.Unlock()
	return s.snapshots.Flush()
}

// commit persists next (journal line + coalesced snapshot) and publishes it; bump says
// whether ProjectService.changed() runs (revision + on_change). Called with mutate held.
func (s *Service) commit(next *pyjson.Object, notice *pyjson.Object, bump bool, source any) error {
	prev := s.current.Load()
	compactNext := project.Compact(next)
	_, size, err := s.encoder.Checksum(compactNext)
	if err != nil {
		return err
	}
	if size > archive.MaxUnpacked {
		return apperr.New("Engineering project exceeds the 100 MB save limit; nothing was written")
	}
	revision := prev.revision
	if bump {
		revision++
	}
	s.seq++
	if err := s.journal.Commit(s.seq, prev.compact, compactNext); err != nil {
		return err
	}
	s.current.Store(&state{project: next, compact: compactNext, notice: notice, revision: revision})
	s.snapshots.Request(s.seq, compactNext)
	if bump {
		s.config.OnChange(revision, source)
	}
	return nil
}
