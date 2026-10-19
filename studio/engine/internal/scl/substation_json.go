package scl

import "studio-engine/internal/pyjson"

// JSON is read_substation's dict with ld_domains written as [[[ied, ldInst], domain], ...]
// (JSON has no tuple keys).
func (t *Topology) JSON() *pyjson.Object {
	substations := make([]any, len(t.Substations))
	for index, name := range t.Substations {
		substations[index] = name
	}
	equipment := make([]any, len(t.Equipment))
	for index, item := range t.Equipment {
		equipment[index] = item.JSON()
	}
	nodes := pyjson.NewObject()
	for _, path := range t.NodeOrder {
		node := t.Nodes[path]
		nodes.Set(path, pyjson.NewObject().Set("path", node.Path).Set("name", node.Name).
			Set("level", int64(node.Level)).Set("bay", int64(node.Bay)).Set("bay_name", node.BayName).
			Set("bus_y", floatOrNone(node.BusY)))
	}
	other := make([]any, len(t.Other))
	for index, item := range t.Other {
		other[index] = pyjson.NewObject().Set("path", item.Path).Set("element", item.Element).Set("type", item.Type)
	}
	domains := make([]any, len(t.DomainOrder))
	for index, key := range t.DomainOrder {
		domains[index] = []any{[]any{key[0], key[1]}, t.Domains[key]}
	}
	diagnostics := make([]any, len(t.Diagnostics))
	for index, text := range t.Diagnostics {
		diagnostics[index] = text
	}
	return pyjson.NewObject().Set("substations", substations).Set("equipment", equipment).Set("nodes", nodes).
		Set("other", other).Set("ld_domains", domains).Set("diagnostics", diagnostics)
}

// JSON is one equipment dict of read_substation.
func (e *Equipment) JSON() *pyjson.Object {
	var esld any
	if e.ESLD != nil {
		esld = pyjson.NewObject().Set("x", e.ESLD[0]).Set("y", e.ESLD[1])
	}
	terminals := make([]any, len(e.Terminals))
	for index, path := range e.Terminals {
		if path != "" {
			terminals[index] = path
		}
	}
	return pyjson.NewObject().Set("path", e.Path).Set("name", e.Name).Set("type", e.Type).
		Set("level", int64(e.Level)).Set("bay", int64(e.Bay)).Set("bay_name", e.BayName).Set("esld", esld).
		Set("lnodes", LNodesJSON(e.LNodes)).Set("terminals", terminals)
}

// LNodesJSON is the lnodes list ({iedName, ldInst, prefix, lnClass, lnInst}).
func LNodesJSON(lnodes []LNode) []any {
	result := make([]any, len(lnodes))
	for index, node := range lnodes {
		result[index] = pyjson.NewObject().Set("iedName", node.IEDName).Set("ldInst", node.LDInst).
			Set("prefix", node.Prefix).Set("lnClass", node.LNClass).Set("lnInst", node.LNInst)
	}
	return result
}

func floatOrNone(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
