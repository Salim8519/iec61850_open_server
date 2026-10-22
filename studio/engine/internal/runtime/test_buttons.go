package runtime

import (
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/plant"
	"studio-engine/internal/runtime/pyval"
)

const realOnly = "{name} is a real device. Test buttons only change IEDs that ScadaBench simulates; " +
	"operate real equipment with its command link."

// EquipmentTest is POST /equipment/<node>/test (simulation/equipment_test.apply_equipment_test):
// write the bound point(s) as if the equipment changed by itself; no command is sent.
func (c *Coordinator) EquipmentTest(nodeID string, action, value, row any) (*pyjson.Object, error) {
	project := c.opt.Project.Current()
	var node *pyjson.Object
	for _, candidate := range diagramNodes(project) {
		if pyval.Str(candidate, "id") == nodeID {
			node = candidate
			break
		}
	}
	if node == nil {
		return nil, apperr.New("Unknown equipment")
	}
	if pyval.Str(pyval.Map(node, "data"), "kind") == pointPanelKind { // ADR-0102
		return c.pointPanelRequest(nodeID, node, action, value, row)
	}
	if !c.isLive() {
		return nil, apperr.New("Run the IED before using a test button")
	}
	kindOf := func(binding plant.Binding) string {
		c.mu.Lock()
		defer c.mu.Unlock()
		if item := c.runs[binding.IedID]; item != nil {
			if info, ok := item.model.lookup(binding.Ref); ok {
				return info.Kind
			}
		}
		return ""
	}
	writes, err := plant.ButtonWrites(pyval.Map(node, "data"), action, value, row, kindOf)
	if err != nil {
		return nil, apperr.New(err.Error())
	}
	var order []string
	grouped := map[string][]item{}
	for _, write := range writes {
		ied := findIed(project, write.Binding.IedID)
		if ied == nil {
			return nil, apperr.Coded("Unknown IED", "not_found")
		}
		name := pyval.Str(ied, "name")
		if roleOf(ied) != "simulated" {
			return nil, apperr.New(strings.Replace(realOnly, "{name}", name, 1))
		}
		target := c.simulatedRun(pyval.Str(ied, "id"))
		if target == nil {
			return nil, apperr.New(name + " is not running. Run this IED first.")
		}
		info, ok := target.model.lookup(write.Binding.Ref)
		if !ok {
			return nil, apperr.KeyError("('" + write.Binding.IedID + "', '" + write.Binding.Ref + "')")
		}
		next, quality := write.Value, write.Quality
		if next == nil { // quality only: keep the present value
			next, _ = target.table.get(write.Binding.Ref)
		}
		if quality != nil && info.QualityRef == "" {
			if pyval.Truthy(quality) {
				return nil, apperr.New("This point has no quality attribute in its SCL, so its quality cannot be set")
			}
			quality = nil
		}
		id := pyval.Str(ied, "id")
		if _, seen := grouped[id]; !seen {
			order = append(order, id)
		}
		grouped[id] = append(grouped[id], item{write.Binding.Ref, next, quality})
	}
	written := pyjson.NewObject()
	for _, id := range order {
		result, err := c.apply(id, grouped[id], "equipment_test")
		if err != nil {
			return nil, err
		}
		written.Set(id, result)
	}
	c.emit("equipment_test", pyval.Obj("node_id", nodeID, "action", action))
	return pyval.Obj("node_id", nodeID, "action", action, "written", written), nil
}
