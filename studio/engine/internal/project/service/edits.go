package service

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
)

// Diagram is ProjectService.diagram(value, expected_revision, change_source); a nil
// expectedRevision skips the check (internal callers).
func (s *Service) Diagram(value any, expectedRevision *int64, changeSource any) error {
	s.mutate.Lock()
	defer s.mutate.Unlock()
	st := s.current.Load()
	if expectedRevision != nil && *expectedRevision != st.revision {
		return apperr.Coded("Another tab changed this project. This tab must synchronize before saving.", "conflict")
	}
	diagram, err := project.ValidateDiagramWith(value, s.index.For(iedList(st.project)))
	if err != nil {
		return err
	}
	next := shallow(st.project)
	next.Set("diagram", diagram)
	if err := s.commit(next, st.notice, true, changeSource); err != nil {
		return err
	}
	s.validated, s.validatedIEDs = diagram, rawIEDs(st.project)
	return nil
}

// FindNode is ProjectService.find_node: StudioError "Unknown equipment" (not_found).
func (s *Service) FindNode(key string) (*pyjson.Object, error) {
	for _, node := range diagramNodes(s.current.Load().project) {
		if id, _ := node.Get("id"); id == key {
			return node, nil
		}
	}
	return nil, apperr.Coded("Unknown equipment", "not_found")
}

func diagramNodes(p *pyjson.Object) []*pyjson.Object {
	raw, _ := p.Get("diagram")
	diagram, _ := raw.(*pyjson.Object)
	if diagram == nil {
		return nil
	}
	list, _ := diagram.Get("nodes")
	items, _ := list.([]any)
	out := []*pyjson.Object{}
	for _, item := range items {
		if node, ok := item.(*pyjson.Object); ok {
			out = append(out, node)
		}
	}
	return out
}

// NodeData is ProjectService.node_data: node data merged, diagram re-validated, saved
// without a revision change (runtime switch positions).
func (s *Service) NodeData(nodeID string, changes *pyjson.Object) error {
	s.mutate.Lock()
	defer s.mutate.Unlock()
	if _, err := s.FindNode(nodeID); err != nil {
		return err
	}
	st := s.current.Load()
	rawDiagram, _ := st.project.Get("diagram")
	current := rawDiagram.(*pyjson.Object)
	lookup := s.index.For(iedList(st.project))
	var diagram *pyjson.Object
	var err error
	if current == s.validated {
		same := sameList(rawIEDs(st.project), s.validatedIEDs)
		diagram, err = project.NodeDataDiagram(current, nodeID, changes, lookup, same)
	} else {
		diagram, err = project.ValidateDiagramWith(nodeDataInput(current, nodeID, changes), lookup)
	}
	if err != nil {
		return err
	}
	next := shallow(st.project)
	next.Set("diagram", diagram)
	if err := s.commit(next, st.notice, false, nil); err != nil {
		return err
	}
	s.validated, s.validatedIEDs = diagram, rawIEDs(st.project)
	return nil
}

// nodeDataInput is node_data's {"nodes": [...updated...], "edges": edges} for a diagram
// that is not known to be validated output (full validation).
func nodeDataInput(current *pyjson.Object, nodeID string, changes *pyjson.Object) *pyjson.Object {
	nodes := []any{}
	for _, node := range diagramNodes(pyjson.NewObject().Set("diagram", current)) {
		if id, _ := node.Get("id"); id == nodeID {
			updated := shallow(node)
			rawData, _ := node.Get("data")
			data := shallow(rawData.(*pyjson.Object))
			for _, key := range changes.Keys() {
				value, _ := changes.Get(key)
				data.Set(key, value)
			}
			node = updated.Set("data", data)
		}
		nodes = append(nodes, node)
	}
	edges, _ := current.Get("edges")
	return pyjson.NewObject().Set("nodes", nodes).Set("edges", edges)
}

// Source is ProjectService.source: replace one signal source and re-validate all (it
// implements runtime.Project.Source).
func (s *Service) Source(iedID, reference string, spec *pyjson.Object) error {
	s.mutate.Lock()
	defer s.mutate.Unlock()
	st := s.current.Load()
	raw, _ := st.project.Get("sources")
	list, _ := raw.([]any)
	sources := []any{}
	for _, item := range list {
		entry := item.(*pyjson.Object)
		id, _ := entry.Get("ied_id")
		ref, _ := entry.Get("ref")
		if id == iedID && ref == reference {
			continue
		}
		sources = append(sources, entry)
	}
	var value any // a nil spec is Python's None, not a typed nil
	if spec != nil {
		value = spec
	}
	sources = append(sources, pyjson.NewObject().Set("ied_id", iedID).Set("ref", reference).Set("spec", value))
	checked, err := project.ValidateSources(sources, iedList(st.project))
	if err != nil {
		return err
	}
	return s.replace("sources", checked, nil)
}

// Scenarios is ProjectService.scenarios.
func (s *Service) Scenarios(value any) error {
	s.mutate.Lock()
	defer s.mutate.Unlock()
	checked, err := project.ValidateScenarios(value, iedList(s.current.Load().project))
	if err != nil {
		return err
	}
	return s.replace("scenarios", checked, nil)
}

// replace sets one top-level key and commits with a revision change. mutate is held.
func (s *Service) replace(key string, value any, source any) error {
	st := s.current.Load()
	next := shallow(st.project)
	next.Set(key, value)
	return s.commit(next, st.notice, true, source)
}

// Update runs a project-rule mutation (project.UpdateIED, UpdateNetwork, SaveReach, ...)
// on a draft copy and commits it with a revision change when it succeeds.
func (s *Service) Update(change func(draft *pyjson.Object) (*pyjson.Object, error)) (*pyjson.Object, error) {
	s.mutate.Lock()
	defer s.mutate.Unlock()
	st := s.current.Load()
	next := draft(st.project)
	result, err := change(next)
	if err != nil {
		return nil, err
	}
	if err := s.commit(next, st.notice, true, nil); err != nil {
		return nil, err
	}
	return result, nil
}
