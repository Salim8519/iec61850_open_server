package service

import (
	"studio-engine/internal/memtrim"
	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/recovery"
)

// RemovalPreview is ProjectService.removal_preview(key).
func (s *Service) RemovalPreview(key string) (*pyjson.Object, error) {
	return project.RemovalPreview(s.current.Load().project, key)
}

// BulkPreview is bulk_removal.preview(service, keys).
func (s *Service) BulkPreview(keys any) (*pyjson.Object, error) {
	return project.BulkPreview(s.current.Load().project, keys)
}

// BulkRemove is bulk_removal.remove: fingerprint check, verified recovery backup of the
// current project first, then the removal is saved and announced. The result is the
// preview plus "recovery_file".
func (s *Service) BulkRemove(keys, fingerprint any) (*pyjson.Object, error) {
	defer memtrim.Soon()
	s.mutate.Lock()
	defer s.mutate.Unlock()
	st := s.current.Load()
	preview, candidate, err := project.BulkRemove(st.project, keys, fingerprint)
	if err != nil {
		return nil, err
	}
	payload, err := s.encoder.Encode(st.compact)
	if err != nil {
		return nil, err
	}
	backup, err := recovery.WriteDeleteBackup(s.config.DataDir, payload)
	if err != nil {
		return nil, err
	}
	if err := s.commit(candidate, st.notice, true, nil); err != nil {
		return nil, err
	}
	return shallow(preview).Set("recovery_file", backup), nil
}

// RemoveIED is ProjectService.remove_ied(key): the single-IED preview, removed through
// the bulk path with its own fingerprint.
func (s *Service) RemoveIED(key string) (*pyjson.Object, error) {
	defer memtrim.Soon()
	preview, err := s.RemovalPreview(key)
	if err != nil {
		return nil, err
	}
	bulk, err := s.BulkPreview([]any{key})
	if err != nil {
		return nil, err
	}
	fingerprint, _ := bulk.Get("fingerprint")
	result, err := s.BulkRemove([]any{key}, fingerprint)
	if err != nil {
		return nil, err
	}
	file, _ := result.Get("recovery_file")
	return shallow(preview).Set("recovery_file", file), nil
}
