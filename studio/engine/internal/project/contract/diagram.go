package contract

import "studio-engine/internal/pyjson"

const hexColor = `^#[0-9a-fA-F]{6}$`

// Equipment kinds; TRANSFORMER/LOAD/BUSBAR were retired in format 12 (ADR-0033),
// INFEED is drawing-only (ADR-0041) and MEASUREMENT_LIST is a Go-only panel (ADR-0090). The
// Go-only POINT_PANEL (ADR-0102) and TAP_TRANSFORMER (ADR-0109) are accepted by kindChoice,
// not listed here.
var equipmentKinds = []string{"CB", "DS", "ES", "CT", "VT", "SOURCE", "RELAY", "TRIP_RELAY",
	"PUSH_BUTTON", "JUNCTION", "INFEED", "MEASUREMENT_LIST"}

// MeasurementLimit caps the rows of one measurement list (like the relay's 50 statuses).
const MeasurementLimit = 50

var equipmentData = model("EquipmentData", true,
	required("label", text(1, 100, "")),
	required("kind", kindChoice),
	defaulted("rotation", integer(false, 0, 0), int64(0)),
	defaulted("state_mapping", optional(stateMapping), nil),
	defaulted("control_binding", optional(controlBinding), nil),
	field{name: "relay_statuses", check: list(50, relayStatus), fallback: func() any { return []any{} }},
	omittable("measurements", list(MeasurementLimit, measurement)),
	omittable("ied_id", optional(text(0, 0, ""))),
	omittable("points", list(PointPanelLimit, pointRow)),
	omittable("tap_changer", tapChanger),
	omittable("command_options", commandOptions),
	defaulted("trip_command_polarity", literal("operate_false", "operate_true"), "operate_false"),
	defaulted("trip_state_polarity", literal("operated_false", "operated_true"), "operated_true"),
	defaulted("push_mode", literal("spc_momentary", "latching_status", "self_resetting"), "spc_momentary"),
	defaulted("push_value", boolean, true),
	defaulted("push_rest_value", boolean, false),
	defaulted("push_reset_ms", integer(true, 100, 60000), int64(1000)),
	defaulted("initial_position", literal("open", "closed"), "open"),
	defaulted("active_color", optional(text(0, 0, hexColor)), nil),
	defaulted("inactive_color", optional(text(0, 0, hexColor)), nil),
	defaulted("infeed_type", optional(literal("line", "transformer", "grid")), nil),
	defaulted("infeed_state", optional(literal("live", "dead")), nil))

var equipmentNode = model("EquipmentNode", false,
	required("id", text(0, 0, identifier)),
	defaulted("type", literal("equipment"), "equipment"),
	required("position", position),
	required("data", equipmentData))

var wire = model("Wire", false,
	required("id", text(0, 128, "")),
	required("source", text(0, 0, "")),
	required("target", text(0, 0, "")),
	defaulted("sourceHandle", optional(text(0, 0, "")), nil),
	defaulted("targetHandle", optional(text(0, 0, "")), nil))

var diagram = model("Diagram", false,
	field{name: "nodes", check: list(2000, equipmentNode), fallback: func() any { return []any{} }},
	field{name: "edges", check: list(5000, wire), fallback: func() any { return []any{} }})

// ValidateDiagram is Diagram.model_validate(value).model_dump(): the dumped diagram with
// every default set, or a *ContractError with pydantic's error list.
func ValidateDiagram(value any) (*pyjson.Object, error) {
	c := &collector{}
	result, ok := diagram(c, loc{}, value)
	if !ok {
		return nil, &ContractError{Details: c.details}
	}
	return result.(*pyjson.Object), nil
}

// ValidateNodeAt validates one equipment node as item index of a diagram's "nodes": the
// node ValidateDiagram would produce there, or the *ContractError it would raise when
// that node is the only invalid part (the model dump is a fixed point of the contract, so
// a diagram whose other parts are validated output only fails on this node).
func ValidateNodeAt(value any, index int) (*pyjson.Object, error) {
	c := &collector{}
	result, ok := equipmentNode(c, loc{"nodes", index}, value)
	if !ok {
		return nil, &ContractError{Details: c.details}
	}
	return result.(*pyjson.Object), nil
}
