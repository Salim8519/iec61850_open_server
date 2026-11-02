package service

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/memtrim"
	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/archive"
	"studio-engine/internal/store/recovery"
)

// Restore is ProjectService.restore(payload, persist): decode, re-import every stored SCL
// source once, migrate and validate, then swap the project in (persisting when asked)
// and announce the change. Nothing changes when any step fails.
func (s *Service) Restore(payload []byte, persist bool) error {
	defer memtrim.Soon()
	raw, err := archive.Decode(payload)
	if err != nil {
		return err
	}
	document, err := project.Expand(raw)
	if err != nil {
		return err
	}
	version, items, err := project.StoredIEDs(document)
	if err != nil {
		return err
	}
	groups, err := project.StoredGroups(items)
	if err != nil {
		return err
	}
	models := map[string]*pyjson.Object{}
	reads := s.readCached(groups)
	for index, group := range groups {
		names, err := group.Names()
		if err != nil {
			return err
		}
		source, ok := group.Source.(string)
		if !ok {
			return apperr.ValueError("Stored SCL source is not text")
		}
		built, err := s.storedModels(source, group.Filename, names, reads[index])
		if err != nil {
			return err
		}
		if err := group.Assign(built, models); err != nil {
			return err
		}
	}
	restored, err := project.PrepareRestore(project.RestoreInput{
		Raw: document, Version: version, Models: models, NewID: s.config.NewID,
		Preserve: func() (string, error) { return recovery.PreserveMigration(s.config.DataDir, payload) },
	})
	if err != nil {
		return err
	}
	s.mutate.Lock()
	defer s.mutate.Unlock()
	s.history.Clear() // another project: its steps do not apply
	if !persist {
		return s.adopt(restored.Project, restored.Notice)
	}
	return s.commitUnrecorded(restored.Project, restored.Notice)
}

// adopt installs a project restored from the autosave itself (startup, persist=False):
// nothing is written, the revision moves as changed() does.
func (s *Service) adopt(next, notice *pyjson.Object) error {
	prev := s.current.Load()
	compactNext := s.compactor.Compact(next)
	if err := s.encoder.Warm(compactNext); err != nil {
		return err
	}
	revision := prev.revision + 1
	s.current.Store(&state{project: next, compact: compactNext, notice: notice, revision: revision})
	s.config.OnChange(revision, nil)
	return nil
}

// RecoverStartup folds a leftover journal and restores autosave.scadabench like
// startup_recovery.recover_startup; the result is the GET /project startup_recovery value.
func (s *Service) RecoverStartup() (*pyjson.Object, error) {
	defer memtrim.Soon()
	result, err := recovery.RecoverStartup(s.config.DataDir, func(payload []byte) error {
		return s.Restore(payload, false)
	})
	if id, ok := s.Project().Get("id"); ok {
		s.recovered.Store(id)
	}
	return result, err
}

// StillRecovered says the open project is the one startup recovery opened: its notice
// ("Recovered the last saved substation") does not speak about a project made or opened later.
func (s *Service) StillRecovered() bool {
	id, _ := s.Project().Get("id")
	return s.recovered.Load() == id
}
