package runtime

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/runtime/signal"
)

// iedModel is the typed view of one project IED, built once per run from the immutable
// project tree: signals get a point id (their index) so values live in arrays (0111 rule 1).
type iedModel struct {
	id, name, role string
	raw            *pyjson.Object
	signals        []signal.Signal
	rawSignals     []*pyjson.Object
	index          map[string]int32
}

func newIedModel(raw *pyjson.Object) *iedModel {
	items := pyval.List(raw, "signals")
	model := &iedModel{id: pyval.Str(raw, "id"), name: pyval.Str(raw, "name"), raw: raw,
		role: roleOf(raw), signals: make([]signal.Signal, 0, len(items)),
		rawSignals: make([]*pyjson.Object, 0, len(items)), index: make(map[string]int32, len(items))}
	for _, item := range items {
		object, ok := item.(*pyjson.Object)
		if !ok {
			continue
		}
		writable, _ := pyval.Get(object, "writable").(bool)
		entry := signal.Signal{Ref: pyval.Str(object, "ref"), Kind: pyval.Str(object, "kind"), FC: pyval.Str(object, "fc"),
			Writable: writable, QualityRef: pyval.Str(object, "quality_ref"), TimestampRef: pyval.Str(object, "timestamp_ref"),
			Enums: enumChoices(object)}
		if _, seen := model.index[entry.Ref]; !seen {
			model.index[entry.Ref] = int32(len(model.signals))
		} else {
			model.index[entry.Ref] = int32(len(model.signals)) // a dict keeps the last duplicate
		}
		model.signals = append(model.signals, entry)
		model.rawSignals = append(model.rawSignals, object)
	}
	return model
}

// enumChoices are a signal's "enums" ({label: ord}, SCL order) for the EnumType write check.
func enumChoices(object *pyjson.Object) []signal.EnumChoice {
	enums := pyval.Map(object, "enums")
	if enums == nil || enums.Len() == 0 {
		return nil
	}
	choices := make([]signal.EnumChoice, 0, enums.Len())
	for _, label := range enums.Keys() {
		if ord, ok := pyval.Get(enums, label).(int64); ok {
			choices = append(choices, signal.EnumChoice{Label: label, Ord: ord})
		}
	}
	return choices
}

// roleOf is ied.get("role", "simulated").
func roleOf(raw *pyjson.Object) string {
	if role, ok := raw.Get("role"); ok {
		text, _ := role.(string)
		return text
	}
	return "simulated"
}

func (m *iedModel) lookup(ref string) (signal.Signal, bool) {
	position, ok := m.index[ref]
	if !ok {
		return signal.Signal{}, false
	}
	return m.signals[position], true
}

// controls returns ied["controls"] as sent to the worker.
func (m *iedModel) controls() []any { return pyval.List(m.raw, "controls") }

// control finds a control entry by reference.
func (m *iedModel) control(ref string) *pyjson.Object {
	for _, item := range m.controls() {
		entry, _ := item.(*pyjson.Object)
		if pyval.Str(entry, "ref") == ref {
			return entry
		}
	}
	return nil
}

// projectIeds lists the IED objects of a project tree.
func projectIeds(project *pyjson.Object) []*pyjson.Object {
	var ieds []*pyjson.Object
	for _, item := range pyval.List(project, "ieds") {
		if ied, ok := item.(*pyjson.Object); ok {
			ieds = append(ieds, ied)
		}
	}
	return ieds
}

func findIed(project *pyjson.Object, id string) *pyjson.Object {
	for _, ied := range projectIeds(project) {
		if pyval.Str(ied, "id") == id {
			return ied
		}
	}
	return nil
}

func diagramNodes(project *pyjson.Object) []*pyjson.Object {
	var nodes []*pyjson.Object
	for _, item := range pyval.List(pyval.Map(project, "diagram"), "nodes") {
		if node, ok := item.(*pyjson.Object); ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}
