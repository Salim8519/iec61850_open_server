package sld

import "studio-engine/internal/pyjson"

// Diagram is {"nodes": [...], "edges": [...]} of the built diagram (shared objects).
func (b *Built) Diagram() *pyjson.Object {
	nodes, edges := b.Nodes, b.Edges
	if nodes == nil {
		nodes = []any{}
	}
	if edges == nil {
		edges = []any{}
	}
	return pyjson.NewObject().Set("nodes", nodes).Set("edges", edges)
}

// DiagnosticsJSON is the diagnostics list.
func (b *Built) DiagnosticsJSON() []any {
	result := make([]any, len(b.Diagnostics))
	for index, text := range b.Diagnostics {
		result[index] = text
	}
	return result
}

// JSON is build_diagram's dict: {diagram, skipped, equipment: {id: item}, diagnostics, layout}.
func (b *Built) JSON() *pyjson.Object {
	equipment := pyjson.NewObject()
	for _, id := range b.EquipmentIDs {
		equipment.Set(id, b.Equipment[id].JSON())
	}
	return pyjson.NewObject().Set("diagram", b.Diagram()).Set("skipped", b.Skipped).Set("equipment", equipment).
		Set("diagnostics", b.DiagnosticsJSON()).Set("layout", b.Layout)
}
