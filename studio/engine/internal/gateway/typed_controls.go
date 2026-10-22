package gateway

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// ExtendedControls supplies an IED's typed controls (BSC, ISC, APC, INC, ENC; ADR-0104). They are
// read from the stored SCL and are not part of the IED payload, so this package cannot see them
// by itself. The app wires it to the SCL reader at start; nil (unit tests) means none.
var ExtendedControls func(ied *pyjson.Object) []any

// TypedControlsOf is the typed control list of an IED (nil when none or not wired).
func TypedControlsOf(ied *pyjson.Object) []any {
	if ExtendedControls == nil || ied == nil {
		return nil
	}
	return ExtendedControls(ied)
}

// typedControlReason is why a step or set point command cannot drive ref ("" when it can): the
// control is looked up in "controls" first, then in the typed list, and its class checked.
func typedControlReason(kind string, ied *pyjson.Object, ref string) (string, error) {
	controls, err := refIndex(ied, "controls", []any{})
	if err != nil {
		return "", err
	}
	control := controls[ref]
	if control == nil {
		for _, item := range TypedControlsOf(ied) {
			if entry, ok := item.(*pyjson.Object); ok && pytext.Str(getOr(entry, "ref", "")) == ref {
				control = entry
			}
		}
	}
	if control == nil {
		name, err := subscript(ied, "name")
		return pytext.Str(name) + " has no controllable object " + ref, err
	}
	return controlClassReason(kind, control, ref), nil
}
