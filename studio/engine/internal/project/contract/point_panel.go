package contract

import "studio-engine/internal/project/pyval"

// The point panel (ADR-0102) is a Go-only panel kind: one IED, up to PointPanelLimit points.
// Its fields are omittable, so documents without a point panel keep their exact bytes.

// pointPanelKind is the kind name; the project package repeats it as PointPanelKind.
const pointPanelKind = "POINT_PANEL"

// PointPanelLimit caps the points of one panel. A bay of a protection relay has about 100 to
// 250 points; 300 fits one bay with room, keeps the saved panel near 100 KB, and stays below
// the 500 points a real IED session may poll (acquisition.WatchedLimit).
const PointPanelLimit = 300

// knownKind is the literal list of the kinds the frozen Python contract knows plus the Go-only
// MEASUREMENT_LIST. It stays unchanged so the text of the unknown-kind error is the one the
// golden vectors record (ADR-0102 lists this as a limitation).
var knownKind = literal(equipmentKinds...)

// kindChoice accepts every kind of equipmentKinds, the point panel and the tap-changer
// transformer (ADR-0109).
func kindChoice(c *collector, at loc, input any) (any, bool) {
	if input == pointPanelKind || input == tapTransformerKind {
		return input, true
	}
	return knownKind(c, at, input)
}

// healthyValue is the optional "healthy" value of a point: on/off, a Dbpos position or a number
// (the raw value the IED holds). The project package checks it against the point's type.
func healthyValue(c *collector, at loc, input any) (any, bool) {
	switch input.(type) {
	case bool:
		return input, true
	case float64:
		return bounded(1e9)(c, at, input)
	}
	if _, whole := pyval.BigInt(input); whole {
		return integer(true, -1000000000, 1000000000)(c, at, input)
	}
	c.add("healthy_type", at, "Input should be true, false or a number")
	return nil, false
}

// pointRow is one point of a panel: a status (BOOLEAN), a double point (Dbpos), a command
// (SPC or DPC control) or a value (numeric MX). Status, double and value rows read `source`;
// a command row uses `control`. unit, scale, offset, decimals and pinned belong to value rows.
var pointRow = model("PointRow", true,
	required("id", text(0, 0, identifier)),
	required("label", text(1, 100, "")),
	defaulted("group", text(0, 100, ""), ""),
	required("type", literal("status", "double", "command", "value")),
	defaulted("source", optional(signalBinding), nil),
	defaulted("control", optional(controlBinding), nil),
	defaulted("unit", text(0, 16, ""), ""),
	defaulted("scale", bounded(1e9), 1.0),
	defaulted("offset", bounded(1e9), 0.0),
	defaulted("decimals", integer(true, 0, 6), int64(2)),
	defaulted("pinned", boolean, false),
	defaulted("healthy", optional(healthyValue), nil))
