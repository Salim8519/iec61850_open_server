package service

import "studio-engine/internal/pyjson"

// SNMP returns the stored SNMP devices section (ADR-0092), or nil when the project has none.
func (s *Service) SNMP() *pyjson.Object {
	value, _ := s.Project().Get("snmp")
	section, _ := value.(*pyjson.Object)
	return section
}

// SaveSNMP stores the SNMP section and saves; on a save failure the previous section stays.
// The revision stays and no project change is announced: the section is not part of the
// project snapshot, so the pages have nothing to reload.
func (s *Service) SaveSNMP(section *pyjson.Object) error {
	s.mutate.Lock()
	defer s.mutate.Unlock()
	st := s.current.Load()
	next := shallow(st.project)
	next.Set("snmp", section)
	return s.commit(next, st.notice, false, nil)
}
