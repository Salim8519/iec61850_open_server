package project

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project/sld"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// projects/sld_import.py: preview a single-line diagram built from a stored SCL Substation
// section. Preview never changes the project; the browser applies the diagram through the
// revision-guarded diagram save (one undo step).

const besideGap = 240.0

// SubstationSources is substation_sources(service): every stored file, saying whether it
// has a substation layout to draw.
func SubstationSources(project *pyjson.Object) ([]any, error) {
	result := []any{}
	for _, entry := range StoredSources(project) {
		topology, err := scl.ReadSubstation(entry.Text)
		if err != nil {
			return nil, err
		}
		substations := make([]any, len(topology.Substations))
		for index, name := range topology.Substations {
			substations[index] = name
		}
		result = append(result, pyjson.NewObject().Set("id", entry.ID).Set("filename", entry.Filename).
			Set("ieds", entry.IEDNames()).Set("substations", substations).Set("equipment", int64(len(topology.Equipment))))
	}
	return result, nil
}

// PreviewDiagram is sld_import.preview(service, source_id, mode) for the project at revision.
func PreviewDiagram(project *pyjson.Object, revision int64, sourceID, mode string) (*pyjson.Object, error) {
	if mode != "replace" && mode != "beside" {
		return nil, apperr.New("Choose replace or beside")
	}
	entry, err := RequireSource(project, sourceID)
	if err != nil {
		return nil, err
	}
	topology, err := scl.ReadSubstation(entry.Text)
	if err != nil {
		return nil, err
	}
	if len(topology.Substations) == 0 {
		return nil, apperr.New("This file has no substation layout. Ask for the SCD from the system integrator.")
	}
	built, err := sld.BuildDiagram(topology, sourceID)
	if err != nil {
		return nil, apperr.New(err.Error())
	}
	ieds := projectIEDs(project)
	drawn, unbound := bindBuilt(built, ieds, topology.Domains)
	current := object(valueOf(project, "diagram"))
	merged, present := mergeDiagram(current, built, mode)
	diagram, err := ValidateDiagram(merged, ieds)
	if err != nil {
		return nil, err
	}
	removed := int64(0)
	if mode == "replace" {
		removed = int64(len(items(valueOf(current, "nodes"))))
	}
	return pyjson.NewObject().Set("diagram", diagram).Set("revision", revision).Set("layout", built.Layout).
		Set("drawn", drawn).Set("skipped", built.Skipped).Set("unbound", unbound).
		Set("diagnostics", built.DiagnosticsJSON()).Set("already_present", present).Set("removed", removed), nil
}

// bindBuilt is _bind: bindings written into the built switch nodes, the drawn list and the
// reasons of what stayed unbound.
func bindBuilt(built *sld.Built, ieds []*pyjson.Object, domains map[[2]string]string) ([]any, []any) {
	drawn, unbound := []any{}, []any{}
	for _, raw := range built.Nodes {
		node := object(raw)
		item := built.Equipment[text(node, "id")]
		if item == nil {
			continue
		}
		data := object(valueOf(node, "data"))
		kind := text(data, "kind")
		state, control, reasons := sld.Bind(kind, item.LNodes, ieds, domains)
		if state != nil {
			data.Set("state_mapping", state)
		}
		if control != nil {
			data.Set("control_binding", control)
		}
		drawn = append(drawn, pyjson.NewObject().Set("path", item.Path).Set("kind", kind).
			Set("label", valueOf(data, "label")).Set("bound", state != nil || control != nil))
		if len(reasons) > 0 {
			listed := make([]any, len(reasons))
			for index, reason := range reasons {
				listed[index] = reason
			}
			unbound = append(unbound, pyjson.NewObject().Set("path", item.Path).Set("kind", kind).Set("reasons", listed))
		}
	}
	return drawn, unbound
}

// mergeDiagram is _merge: the built diagram alone (replace), or added to the right of the
// current one without the nodes and wires it already has (beside).
func mergeDiagram(current *pyjson.Object, built *sld.Built, mode string) (*pyjson.Object, []any) {
	if mode == "replace" {
		return built.Diagram(), []any{}
	}
	nodes := append([]any{}, items(valueOf(current, "nodes"))...)
	existing := map[string]bool{}
	var right *float64
	for _, raw := range nodes {
		node := object(raw)
		existing[text(node, "id")] = true
		if x, ok := numberOf(valueOf(object(valueOf(node, "position")), "x")); ok && (right == nil || x > *right) {
			right = &x
		}
	}
	shift := 0.0
	if right != nil {
		lowest := 0.0
		for index, raw := range built.Nodes {
			if x, _ := numberOf(valueOf(object(valueOf(object(raw), "position")), "x")); index == 0 || x < lowest {
				lowest = x
			}
		}
		shift = *right + besideGap - lowest
	}
	present := []any{}
	for _, raw := range built.Nodes {
		node := object(raw)
		if existing[text(node, "id")] {
			present = append(present, valueOf(object(valueOf(node, "data")), "label"))
			continue
		}
		position := object(valueOf(node, "position"))
		x, _ := numberOf(valueOf(position, "x"))
		node.Set("position", pyjson.NewObject().Set("x", x+shift).Set("y", valueOf(position, "y")))
		nodes = append(nodes, node)
	}
	wires := map[string]bool{}
	edges := append([]any{}, items(valueOf(current, "edges"))...)
	for _, raw := range edges {
		wires[text(object(raw), "id")] = true
	}
	for _, raw := range built.Edges {
		if !wires[text(object(raw), "id")] {
			edges = append(edges, raw)
		}
	}
	return pyjson.NewObject().Set("nodes", nodes).Set("edges", edges), present
}

func numberOf(value any) (float64, bool) {
	switch v := value.(type) {
	case int64:
		return float64(v), true
	case float64:
		return v, true
	}
	return 0, false
}
