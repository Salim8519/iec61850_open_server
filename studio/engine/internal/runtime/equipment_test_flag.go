package runtime

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// TestCommandMessage is the answer to an equipment command sent with Test=TRUE (change 0176).
const TestCommandMessage = "Test command accepted; equipment not moved"

// testCommandDone finishes an equipment command that carried Test=TRUE (ADR-0108 options). A
// logical node in normal mode does not act on a test command (IEC 61850-7-4 Annex A), and the
// simulated IED ignores it, so the simulated plant feedback and the remembered start position
// are left alone; only the sent options are remembered. Go-only: Python has no options.
func (c *Coordinator) testCommandDone(nodeID string, plan commandPlan, result *pyjson.Object,
	options CommandOptions, sent bool) (*pyjson.Object, error) {
	controlRef := pyval.Str(plan.control, "ref")
	c.emit("equipment_test_command", pyval.Obj("node_id", nodeID, "control_ref", controlRef, "action", plan.semantic,
		"wire_value", plan.wire, "origin", "studio_api", "message", TestCommandMessage))
	if remembered := optionChanges(plan.data, options, sent); remembered != nil {
		if err := c.opt.Project.NodeData(nodeID, remembered); err != nil {
			return nil, err
		}
	}
	return pyval.Obj("command", result, "feedback", []any{}, "node_id", nodeID, "action", plan.semantic,
		"wire_value", plan.wire, "message", TestCommandMessage), nil
}
