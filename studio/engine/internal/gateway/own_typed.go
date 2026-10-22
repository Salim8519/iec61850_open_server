package gateway

import (
	"math"

	"studio-engine/internal/tools/pytext"
)

// Own points of the typed kinds (ADR-0107, Go only): a tap position the user sets, a step
// command that moves an own tap position by one, and a set point that sets an own value.

// typedOwnValue checks a value for an own point of a typed kind.
func typedOwnValue(kind string, raw any) (any, string) {
	_, isBool := raw.(bool)
	switch kind {
	case KindStepPosition:
		if !isBool && pytext.IntIn(raw, -64, 63) {
			number, _ := pytext.Int64(raw)
			return number, ""
		}
		return nil, "the tap position must be a whole number from -64 to 63"
	case KindStep:
		if raw == "higher" || raw == "lower" {
			return raw, ""
		}
		return nil, "the step must be higher (raise) or lower"
	case KindSetpoint:
		number, ok := pytext.Float(raw)
		if isBool || !ok || math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, "the set point must be a number"
		}
		return number, ""
	}
	return nil, "choose a type"
}

// targetKinds are the own point kinds each own command may drive.
var targetKinds = map[string][]string{
	"command_single": {"single", "double"}, "command_double": {"single", "double"},
	KindStep:     {KindStepPosition, "integer"},
	KindSetpoint: {"float", "integer", KindStepPosition},
}

// targetReason is why an own command cannot drive the point target of kind ("" when it can).
func targetReason(command, kind, target string) string {
	if contains(targetKinds[command], kind) {
		return ""
	}
	switch command {
	case KindStep:
		return "the point it moves (" + target + ") is not an own tap position or whole number point of this list"
	case KindSetpoint:
		return "the point it sets (" + target + ") is not an own measured value, whole number or tap position of this list"
	}
	return "the status point it changes (" + target + ") is not an own single or double point of this list"
}
