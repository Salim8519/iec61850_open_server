package project

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/goosemsg"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// IEDComponentKind is one whole IED on the SLD (ADR-0112): its name, role, run state and GOOSE
// messages. It has no terminals and no bindings; `ied_id` (null until chosen) names the IED.
// Python does not know this kind.
const IEDComponentKind = goosemsg.IEDKind

// checkIedComponent checks one IED component: no single-signal settings, and a known IED once
// one is chosen. Messages say what to change.
func checkIedComponent(data *pyjson.Object, kind string, ieds *Lookup) error {
	if kind != IEDComponentKind {
		return nil
	}
	if pyval.Truthy(pyval.Lookup(data, "state_mapping", nil)) {
		return apperr.New("An IED component shows a whole IED, so it has no state mapping. " +
			"Use a breaker or a point panel for one signal")
	}
	if pyval.Truthy(pyval.Lookup(data, "control_binding", nil)) {
		return apperr.New("An IED component has no command. Its GOOSE messages are sent from its own sheet")
	}
	if id := text(data, "ied_id"); id != "" && ieds.IED(id) == nil {
		return apperr.New("This IED component shows an IED that is not in the project. Choose another IED " +
			"or import it again")
	}
	return nil
}

// checkUniqueIEDComponents allows one component per IED on the drawing, whatever was changed.
func checkUniqueIEDComponents(nodes []any, ieds *Lookup) error {
	seen := map[string]bool{}
	for _, node := range nodes {
		data := object(pyval.Lookup(object(node), "data", nil))
		id := text(data, "ied_id")
		if text(data, "kind") != IEDComponentKind || id == "" {
			continue
		}
		if seen[id] {
			name := text(ieds.IED(id), "name")
			if name == "" {
				name = "This IED"
			}
			return apperr.New(name + " already has an IED component on this drawing. One component per IED: " +
				"use the one that is there")
		}
		seen[id] = true
	}
	return nil
}

// settleIedComponent gives a fresh IED component no IED yet; it never carries point rows.
func settleIedComponent(data *pyjson.Object) {
	data.Delete("points")
	if _, present := data.Get("ied_id"); !present {
		data.Set("ied_id", nil)
	}
}

// withoutIEDComponents removes the components of removed IEDs (they have no wires; any wire
// that touches one goes with it).
func withoutIEDComponents(diagram *pyjson.Object, ids map[string]bool) {
	gone := map[string]bool{}
	kept := []any{}
	for _, node := range items(pyval.Lookup(diagram, "nodes", nil)) {
		data := object(pyval.Lookup(object(node), "data", nil))
		if kindOf(data) == IEDComponentKind && ids[text(data, "ied_id")] {
			gone[text(object(node), "id")] = true
			continue
		}
		kept = append(kept, node)
	}
	if len(gone) == 0 {
		return
	}
	edges := []any{}
	for _, edge := range items(pyval.Lookup(diagram, "edges", nil)) {
		if !gone[text(object(edge), "source")] && !gone[text(object(edge), "target")] {
			edges = append(edges, edge)
		}
	}
	diagram.Set("nodes", kept).Set("edges", edges)
}
