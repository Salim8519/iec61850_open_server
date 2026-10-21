package runtime

import (
	"sync"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// MemoryBus keeps every event in memory with runtime/event_bus.py's envelope
// ({"id", "kind", "timestamp", **payload}); it stands in for stream A's SSE ring in tests and
// in the parity recorder.
type MemoryBus struct {
	mu     sync.Mutex
	events []*pyjson.Object
}

// Publish implements Bus.
func (b *MemoryBus) Publish(kind string, payload *pyjson.Object) *pyjson.Object {
	b.mu.Lock()
	defer b.mu.Unlock()
	event := pyval.Obj("id", int64(len(b.events)+1), "kind", kind,
		"timestamp", float64(time.Now().UnixNano())/1e9)
	for _, key := range payload.Keys() {
		value, _ := payload.Get(key)
		event.Set(key, value)
	}
	b.events = append(b.events, event)
	return event
}

// Events returns a copy of the published events.
func (b *MemoryBus) Events() []*pyjson.Object {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]*pyjson.Object(nil), b.events...)
}

// MemoryProject is an in-memory Project over one project tree (tests, parity recorder and
// the Go preview until stream B's project actor is merged). Mutations replace the tree.
type MemoryProject struct {
	mu       sync.Mutex
	project  *pyjson.Object
	Saves    int
	OnChange func()
}

// NewMemoryProject wraps a decoded project tree.
func NewMemoryProject(project *pyjson.Object) *MemoryProject { return &MemoryProject{project: project} }

// Current implements Project.
func (m *MemoryProject) Current() *pyjson.Object {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.project
}

// NodeData implements Project (ProjectService.node_data without diagram re-validation).
func (m *MemoryProject) NodeData(nodeID string, changes *pyjson.Object) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	next := pyval.Copy(m.project).(*pyjson.Object)
	for _, node := range diagramNodes(next) {
		if pyval.Str(node, "id") != nodeID {
			continue
		}
		data := pyval.Map(node, "data")
		for _, key := range changes.Keys() {
			value, _ := changes.Get(key)
			data.Set(key, value)
		}
		m.project = next
		m.Saves++
		return nil
	}
	return apperr.Coded("Unknown equipment", "not_found")
}

// Source implements Project (ProjectService.source without validate_sources).
func (m *MemoryProject) Source(iedID, ref string, spec *pyjson.Object) error {
	m.mu.Lock()
	next := pyval.Copy(m.project).(*pyjson.Object)
	sources := []any{}
	for _, entry := range pyval.List(next, "sources") {
		object := entry.(*pyjson.Object)
		if pyval.Str(object, "ied_id") != iedID || pyval.Str(object, "ref") != ref {
			sources = append(sources, object)
		}
	}
	sources = append(sources, pyval.Obj("ied_id", iedID, "ref", ref, "spec", pyval.Copy(spec)))
	next.Set("sources", sources)
	m.project = next
	m.Saves++
	onChange := m.OnChange
	m.mu.Unlock()
	if onChange != nil {
		onChange()
	}
	return nil
}

// AssignIP implements Project.
func (m *MemoryProject) AssignIP(iedID, ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ied := findIed(m.project, iedID); ied != nil {
		next := pyval.Copy(m.project).(*pyjson.Object)
		findIed(next, iedID).Set("ip", ip)
		m.project = next
	}
}
