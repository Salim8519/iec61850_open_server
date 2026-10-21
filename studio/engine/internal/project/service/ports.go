package service

import (
	"studio-engine/internal/pyjson"
)

// The views other services declare on the project actor (their port interfaces):
// runtime.Project (Current, NodeData, Source, AssignIP), gateway/service.Projects (IEDs,
// Gateway, SaveGateway) and api/toolsapi.Projects (TestPanel, SaveTestPanel). Getters read
// the published immutable project; savers commit through the mutation lock and, when the
// save fails, keep the previous value (Python restored it in an except block).

// Current is the immutable project tree (runtime.Project).
func (s *Service) Current() *pyjson.Object { return s.Project() }

// IEDs returns the project's IED models (immutable; never mutate them).
func (s *Service) IEDs() []*pyjson.Object { return iedList(s.Project()) }

// Gateway returns the stored gateway section, or nil when it is not an object.
func (s *Service) Gateway() *pyjson.Object {
	value, _ := s.Project().Get("gateway")
	section, _ := value.(*pyjson.Object)
	return section
}

// SaveGateway is ProjectService.gateway(config): stored as given, saved, changed("gateway").
func (s *Service) SaveGateway(config *pyjson.Object) error {
	s.mutate.Lock()
	defer s.mutate.Unlock()
	var value any // a nil section is Python's None
	if config != nil {
		value = config
	}
	return s.replace("gateway", value, "gateway")
}

// TestPanel returns the saved Signal Test controls (immutable).
func (s *Service) TestPanel() []any {
	value, _ := s.Project().Get("test_panel")
	entries, _ := value.([]any)
	return entries
}

// SaveTestPanel is the end of ProjectService.test_panel for an already validated list
// (toolsapi runs validate_panel first): stored, saved, changed().
func (s *Service) SaveTestPanel(entries []any) error {
	s.mutate.Lock()
	defer s.mutate.Unlock()
	if entries == nil {
		entries = []any{}
	}
	return s.replace("test_panel", entries, nil)
}

// AssignIP stores the loopback address the runtime gave an IED without one at Run start
// (networking/loopback.assign_endpoints mutated ied["ip"] in the project). Like Python it
// changes neither the revision nor announces project_changed; unlike Python's "saved with
// the next save" it is journaled at once. An unknown IED or an unchanged address is a
// no-op; a failed save keeps the project unchanged and is reported to OnSnapshotError.
func (s *Service) AssignIP(iedID, ip string) {
	if err := s.assignIP(iedID, ip); err != nil && s.config.OnSnapshotError != nil {
		s.config.OnSnapshotError(err)
	}
}

func (s *Service) assignIP(iedID, ip string) error {
	s.mutate.Lock()
	defer s.mutate.Unlock()
	st := s.current.Load()
	list := rawIEDs(st.project)
	for index, item := range list {
		ied, ok := item.(*pyjson.Object)
		if !ok {
			continue
		}
		if id, _ := ied.Get("id"); id != iedID {
			continue
		}
		if current, _ := ied.Get("ip"); current == ip {
			return nil
		}
		ieds := append([]any(nil), list...)
		ieds[index] = shallow(ied).Set("ip", ip)
		next := shallow(st.project).Set("ieds", ieds)
		return s.commit(next, st.notice, false, nil)
	}
	return nil
}
