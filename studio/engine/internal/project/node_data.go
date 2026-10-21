package project

import (
	"studio-engine/internal/project/contract"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// NodeDataDiagram is the diagram of ProjectService.node_data:
//
//	updated = {**node, "data": {**node["data"], **changes}}
//	nodes = [updated if item["id"] == node_id else item for item in nodes]
//	validate_diagram({"nodes": nodes, "edges": edges}, ieds)
//
// current must be a diagram returned by ValidateDiagram/ValidateDiagramWith/NodeDataDiagram
// (validated output). The contract's model dump is a fixed point of the contract, so only
// the replaced nodes go through it again; the cross-checks (identifiers, wires, bindings
// against the IEDs) run over the whole diagram in Python's order. The result and
// every error are those of a full validation; published objects are never written.
// sameIEDs says current was validated against these very IEDs (the same immutable list):
// the per-node binding checks of the unchanged nodes then cannot fail and are skipped.
func NodeDataDiagram(current *pyjson.Object, nodeID string, changes *pyjson.Object, ieds *Lookup,
	sameIEDs bool) (*pyjson.Object, error) {
	previous := items(pyval.Lookup(current, "nodes", nil))
	nodes := make([]any, len(previous))
	fresh := map[int]bool{}
	var details []any
	for index, item := range previous {
		node := object(item)
		if text(node, "id") != nodeID {
			nodes[index] = item
			continue
		}
		checked, err := contract.ValidateNodeAt(mergedNode(node, changes), index)
		if err != nil {
			problem, isContract := err.(*ContractError)
			if !isContract {
				return nil, err
			}
			details = append(details, problem.Details...)
			continue
		}
		nodes[index] = checked
		fresh[index] = true
	}
	if details != nil {
		return nil, &ContractError{Details: details}
	}
	edges, _ := current.Get("edges")
	diagram := pyjson.NewObject().Set("nodes", nodes).Set("edges", edges)
	return crossCheck(diagram, ieds, fresh, sameIEDs)
}

// mergedNode is {**node, "data": {**node["data"], **changes}} without touching node.
func mergedNode(node, changes *pyjson.Object) *pyjson.Object {
	updated := pyjson.NewObject()
	for _, key := range node.Keys() {
		value, _ := node.Get(key)
		updated.Set(key, value)
	}
	data := pyjson.NewObject()
	if previous := object(pyval.Lookup(node, "data", nil)); previous != nil {
		for _, key := range previous.Keys() {
			value, _ := previous.Get(key)
			data.Set(key, value)
		}
	}
	for _, key := range changes.Keys() {
		value, _ := changes.Get(key)
		data.Set(key, value)
	}
	return updated.Set("data", data)
}
