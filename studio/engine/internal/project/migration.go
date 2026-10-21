package project

import (
	"strconv"
	"strings"

	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// MigrateDiagram is mapping_migration.migrate_diagram: a deep copy of the stored diagram
// converted from archive format version to the current one, with plain-language notices.
// Malformed stored data raises the TypeError/AttributeError CPython raises.
func MigrateDiagram(value any, ieds []*pyjson.Object, version int64) (any, []any, error) {
	m := &migration{result: pyval.DeepCopy(value), ieds: ieds, notices: []any{}}
	if err := m.standardCommands(); err != nil {
		return nil, nil, err
	}
	steps := []struct {
		run   bool
		apply func() error
		stop  bool
	}{
		{version < 12, m.retireLegacy, version >= 10},
		{true, m.buttonEdges, version >= 9},
		{true, m.relayEntries, false},
		{version <= 3, m.flatMappings, false},
		{version < 7, func() error { return m.tripPolarity(version) }, false},
	}
	for _, step := range steps {
		if step.run {
			if err := step.apply(); err != nil {
				return nil, nil, err
			}
		}
		if step.stop {
			break
		}
	}
	return m.result, m.notices, nil
}

type migration struct {
	result  any
	ieds    []*pyjson.Object
	notices []any
}

func (m *migration) notice(text string) { m.notices = append(m.notices, text) }

// list is iter(result.get(key, [])).
func (m *migration) list(key string) ([]any, error) {
	value, err := pyval.Get(m.result, key, []any{})
	if err != nil {
		return nil, err
	}
	return pyval.Iter(value)
}

// nodeData is node.get("data", {}) followed by a check that the data has .get.
func nodeData(node any) (*pyjson.Object, error) {
	data, err := pyval.Get(node, "data", pyjson.NewObject())
	if err != nil {
		return nil, err
	}
	if _, err := pyval.Get(data, "kind", nil); err != nil {
		return nil, err
	}
	return data.(*pyjson.Object), nil
}

// labelOf is data.get("label", node.get("id", fallback)) written with str().
func labelOf(data *pyjson.Object, node any, fallback any) string {
	id, _ := pyval.Get(node, "id", fallback)
	return pyval.Str(pyval.Lookup(data, "label", id))
}

func (m *migration) standardCommands() error {
	nodes, err := m.list("nodes")
	if err != nil {
		return err
	}
	for _, node := range nodes {
		data, err := nodeData(node)
		if err != nil {
			return err
		}
		binding := pyval.Lookup(data, "control_binding", nil)
		if !pyval.Truthy(binding) {
			continue
		}
		compatibility, err := pyval.Pop(binding, "compatibility", "standard")
		if err != nil {
			return err
		}
		if compatibility != "standard" {
			m.notice(labelOf(data, node, "equipment") + ": the CONCEN command mode was removed; commands now " +
				"follow standard IEC 61850 (Close = true, moves on Operate)")
		}
	}
	return nil
}

// memberOf is `v in values` for a Python set built from values (v must be hashable).
func memberOf(v any, values []any) (bool, error) {
	if err := pyval.CheckHashable(v); err != nil {
		return false, err
	}
	return pyval.InValues(v, values), nil
}

func (m *migration) retireLegacy() error {
	nodes, err := m.list("nodes")
	if err != nil {
		return err
	}
	var retired, gone, kept []any
	for _, node := range nodes {
		data, err := nodeData(node)
		if err != nil {
			return err
		}
		if retiredKind, err := pyval.InStrings(pyval.Lookup(data, "kind", nil), "TRANSFORMER", "LOAD", "BUSBAR"); err != nil {
			return err
		} else if retiredKind {
			retired = append(retired, node)
		}
	}
	for _, node := range retired {
		id, _ := pyval.Get(node, "id", nil)
		if err := pyval.CheckHashable(id); err != nil {
			return err
		}
		gone = append(gone, id)
	}
	for _, node := range nodes {
		id, _ := pyval.Get(node, "id", nil)
		if removed, err := memberOf(id, gone); err != nil {
			return err
		} else if !removed {
			kept = append(kept, node)
		}
	}
	m.result.(*pyjson.Object).Set("nodes", append([]any{}, kept...))
	for _, node := range kept {
		data, _ := nodeData(node)
		data.Delete("width")
	}
	edges, err := m.list("edges")
	if err != nil {
		return err
	}
	keptEdges := []any{}
	for _, edge := range edges {
		keep, err := edgeAvoids(edge, gone)
		if err != nil {
			return err
		}
		if keep {
			keptEdges = append(keptEdges, edge)
		}
	}
	m.result.(*pyjson.Object).Set("edges", keptEdges)
	for _, node := range retired {
		data, _ := nodeData(node)
		id, _ := pyval.Get(node, "id", nil)
		kind := strings.ToLower(pyval.Lookup(data, "kind", "").(string))
		m.notice(pyval.Str(pyval.Lookup(data, "label", id)) + ": old " + kind +
			" symbol removed (no longer supported); its wires were removed too")
	}
	return nil
}

// edgeAvoids is `edge.get("source") not in ids and edge.get("target") not in ids`.
func edgeAvoids(edge any, ids []any) (bool, error) {
	for _, end := range []string{"source", "target"} {
		value, err := pyval.Get(edge, end, nil)
		if err != nil {
			return false, err
		}
		if member, err := memberOf(value, ids); err != nil || member {
			return false, err
		}
	}
	return true, nil
}

func (m *migration) buttonEdges() error {
	nodes, err := m.list("nodes")
	if err != nil {
		return err
	}
	var buttons []any
	for _, node := range nodes {
		data, err := nodeData(node)
		if err != nil {
			return err
		}
		if pyval.Lookup(data, "kind", nil) == "PUSH_BUTTON" {
			id, _ := pyval.Get(node, "id", nil)
			if err := pyval.CheckHashable(id); err != nil {
				return err
			}
			buttons = append(buttons, id)
		}
	}
	if len(buttons) == 0 {
		return nil
	}
	edges, err := m.list("edges")
	if err != nil {
		return err
	}
	kept := []any{}
	for _, edge := range edges {
		keep, err := edgeAvoids(edge, buttons)
		if err != nil {
			return err
		}
		if keep {
			kept = append(kept, edge)
		}
	}
	if removed := len(edges) - len(kept); removed > 0 {
		m.result.(*pyjson.Object).Set("edges", kept)
		plural := "s"
		if removed == 1 {
			plural = ""
		}
		m.notice(strconv.Itoa(removed) + " wire" + plural + " to push buttons removed: buttons no longer accept connections")
	}
	return nil
}
