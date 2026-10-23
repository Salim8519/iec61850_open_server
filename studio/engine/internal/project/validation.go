package project

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project/contract"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// ContractError is pydantic's ValidationError for the browser diagram contract.
type ContractError = contract.ContractError

var (
	switches    = []string{"CB", "DS", "ES"}
	drawingOnly = map[string]string{"JUNCTION": "A junction", "INFEED": "An infeed"}
	numericKind = []string{"INT8", "INT16", "INT32", "INT64", "INT128", "INT8U", "INT16U", "INT24U", "INT32U",
		"FLOAT32", "FLOAT64"}
)

func isSwitch(kind string) bool { return pyval.InTuple(kind, switches...) }

// text reads a str field of a validated or engineering object ("" when absent or not str).
func text(object *pyjson.Object, key string) string {
	if object == nil {
		return ""
	}
	value, _ := pyval.Lookup(object, key, nil).(string)
	return value
}

func object(value any) *pyjson.Object {
	result, _ := value.(*pyjson.Object)
	return result
}

func items(value any) []any {
	result, _ := value.([]any)
	return result
}

// ValidateDiagram is validate_diagram: the browser diagram checked against the pydantic
// contract and the project IEDs; it returns the normalised diagram.
func ValidateDiagram(value any, ieds []*pyjson.Object) (*pyjson.Object, error) {
	return ValidateDiagramWith(value, LinearLookup(ieds))
}

// ValidateDiagramWith is ValidateDiagram answering IED/signal/control questions from ieds
// (an Index lookup makes it independent of the number of points).
func ValidateDiagramWith(value any, ieds *Lookup) (*pyjson.Object, error) {
	diagram, err := contract.ValidateDiagram(value)
	if err != nil {
		return nil, err
	}
	return crossCheck(diagram, ieds, nil, false)
}

// crossCheck is the part of validate_diagram after the contract. fresh says which nodes
// came out of the contract in this call (nil: all); the others are earlier validated
// output, whose infeed fields are already settled and which must not be written to.
// onlyFresh skips the per-node checks of the others (they passed against the same IEDs).
func crossCheck(diagram *pyjson.Object, ieds *Lookup, fresh map[int]bool, onlyFresh bool) (*pyjson.Object, error) {
	nodes, edges := items(pyval.Lookup(diagram, "nodes", nil)), items(pyval.Lookup(diagram, "edges", nil))
	ids := map[string]bool{}
	for _, node := range nodes {
		ids[text(object(node), "id")] = true
	}
	if len(ids) != len(nodes) {
		return nil, apperr.New("Equipment identifiers must be unique")
	}
	wires := map[string]bool{}
	for _, edge := range edges {
		wires[text(object(edge), "id")] = true
	}
	if len(wires) != len(edges) {
		return nil, apperr.New("Wire identifiers must be unique")
	}
	for _, edge := range edges {
		if !ids[text(object(edge), "source")] || !ids[text(object(edge), "target")] {
			return nil, apperr.New("A wire references missing equipment")
		}
	}
	if err := checkUniqueIEDComponents(nodes, ieds); err != nil {
		return nil, err
	}
	for index, node := range nodes {
		isFresh := fresh == nil || fresh[index]
		if onlyFresh && !isFresh {
			continue
		}
		data := object(pyval.Lookup(object(node), "data", nil))
		kind := text(data, "kind")
		if kind != "RELAY" && pyval.Truthy(pyval.Lookup(data, "relay_statuses", nil)) {
			return nil, apperr.New("Only a relay accepts named status entries")
		}
		if kind == "RELAY" {
			if err := validateRelay(data, ieds); err != nil {
				return nil, err
			}
		}
		if err := checkMeasurements(data, kind, ieds); err != nil {
			return nil, err
		}
		if err := checkPointPanel(data, kind, ieds); err != nil {
			return nil, err
		}
		if err := checkTapChanger(data, kind, ieds); err != nil {
			return nil, err
		}
		if err := checkIedComponent(data, kind, ieds); err != nil {
			return nil, err
		}
		if err := checkGooseLink(data, kind, ieds); err != nil {
			return nil, err
		}
		if err := checkCommandOptions(data, kind); err != nil {
			return nil, err
		}
		if isFresh {
			if err := settleInfeed(data, kind); err != nil {
				return nil, err
			}
			settleMeasurements(data, kind)
			settlePointPanel(data, kind)
			settleTapChanger(data, kind)
			settleGooseLink(data)
		}
		if err := validateState(data, kind, ieds); err != nil {
			return nil, err
		}
		if err := validateControl(data, kind, ieds); err != nil {
			return nil, err
		}
	}
	return diagram, nil
}

// settleInfeed gives infeeds their defaults; other kinds never carry infeed fields.
func settleInfeed(data *pyjson.Object, kind string) error {
	if kind == "INFEED" {
		data.Set("infeed_type", pyval.Or(pyval.Lookup(data, "infeed_type", nil), "line"))
		data.Set("infeed_state", pyval.Or(pyval.Lookup(data, "infeed_state", nil), "live"))
		return nil
	}
	if pyval.Lookup(data, "infeed_type", nil) != nil || pyval.Lookup(data, "infeed_state", nil) != nil {
		return apperr.New("Only an infeed accepts infeed settings")
	}
	data.Delete("infeed_type")
	data.Delete("infeed_state")
	return nil
}

func validateRelay(data *pyjson.Object, ieds *Lookup) error {
	seen := map[string]bool{}
	for _, item := range items(pyval.Or(pyval.Lookup(data, "relay_statuses", nil), []any{})) {
		entry := object(item)
		id := text(entry, "id")
		if seen[id] {
			return apperr.New("Relay status identifiers must be unique")
		}
		seen[id] = true
		if _, err := bindingSignal(object(pyval.Lookup(entry, "source", nil)), ieds); err != nil {
			return err
		}
	}
	return nil
}

func findByRef(list any, ref string) *pyjson.Object {
	for _, item := range items(list) {
		if entry := object(item); text(entry, "ref") == ref {
			return entry
		}
	}
	return nil
}

func bindingSignal(binding *pyjson.Object, ieds *Lookup) (*pyjson.Object, error) {
	ied := ieds.IED(text(binding, "ied_id"))
	if ied == nil {
		return nil, apperr.New("A signal binding references an unknown IED")
	}
	signal := ieds.Entry(pyval.Lookup(ied, "signals", nil), text(binding, "ref"))
	if signal == nil {
		return nil, apperr.New("A signal binding references an unknown signal")
	}
	if report := pyval.Lookup(binding, "report_ref", nil); pyval.Truthy(report) {
		if err := validateMapping(ied, text(binding, "ref"), report); err != nil {
			return nil, err
		}
	}
	return signal, nil
}
