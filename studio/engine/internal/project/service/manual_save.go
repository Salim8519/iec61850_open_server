package service

import "studio-engine/internal/pyjson"

// Auto save off (ADR-0139): changes stay in the open project in memory, and nothing reaches
// autosave.scadabench or its journal until Save. The next start opens the last saved project.
// Turning auto save back on saves first, so the journal always extends the file on disk.

// ManualSave says auto save is off.
func (s *Service) ManualSave() bool { return s.manual.Load() }

// Unsaved says the open project has changes that Save has not written (auto save off only).
func (s *Service) Unsaved() bool { return s.unsaved.Load() }

// SetAutoSave turns auto save on or off. Turning it on writes the unsaved changes first, under
// the mutation lock so no change slips between that file and the journal that extends it; when
// the write fails, auto save stays off and nothing is lost from the open project.
func (s *Service) SetAutoSave(on bool) error {
	s.mutate.Lock()
	defer s.mutate.Unlock()
	if on && s.unsaved.Load() {
		s.seq++
		s.snapshots.Request(s.seq, s.current.Load().compact)
		if err := s.snapshots.Flush(); err != nil {
			return err
		}
		s.unsaved.Store(false)
	}
	s.manual.Store(!on)
	return nil
}

// persist journals and snapshots a commit, or only marks it unsaved while auto save is off.
// mutate is held.
func (s *Service) persist(prev, next *pyjson.Object) error {
	if s.manual.Load() {
		s.unsaved.Store(true)
		return nil
	}
	if err := s.journal.Commit(s.seq, prev, next); err != nil {
		return err
	}
	s.snapshots.Request(s.seq, next)
	return nil
}
