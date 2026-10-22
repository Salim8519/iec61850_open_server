package gateway

import (
	"strings"

	"studio-engine/internal/pyjson"
)

// Typed gateway kinds (ADR-0107, Go only; the frozen Python engine has none). They join the
// six Python kinds, which keep their order and texts in Kinds:
//   - step_position: a tap position (BSC valWTr.posVal): IEC 104 M_ST (5 / 32), DNP3 analog
//     input, Modbus input register (int16).
//   - command_step: a regulating step (lower / raise) to a BSC or ISC control: IEC 104
//     C_RC (47 / 60), DNP3 CROB (trip or OFF = lower, close or ON = raise).
//   - command_setpoint: a set point to an APC or INC control: IEC 104 C_SE (48-50 / 61-63),
//     DNP3 analog output (g41 var 1-3), Modbus holding register (FC6 / FC16).
const (
	KindStepPosition = "step_position"
	KindStep         = "command_step"
	KindSetpoint     = "command_setpoint"
)

// TypedKinds are the kinds that only the Go engine serves.
var TypedKinds = []string{KindStepPosition, KindStep, KindSetpoint}

// KnownKind reports a kind of Kinds or TypedKinds.
func KnownKind(kind any) bool { return contains(Kinds, kind) || contains(TypedKinds, kind) }

// IsTypedKind reports a Go-only kind.
func IsTypedKind(kind string) bool { return contains(TypedKinds, kind) }

// SetpointFormats are the wire formats of a set point per protocol; the first is the default.
var SetpointFormats = map[string][]string{
	"iec104": {"float", "scaled", "normalized"}, // C_SE_NC_1 50, C_SE_NB_1 49, C_SE_NA_1 48
	"dnp3":   {"float", "int16", "int32"},       // g41 var 3, var 2, var 1
	"modbus": {"float", "int16", "uint16", "int32"},
}

// WordOrders are the Modbus orders of a 32-bit value's two registers; the first is the default.
var WordOrders = []string{"high_first", "low_first"}

// stepClasses and setpointClasses are the IEC 61850 control classes each command kind drives.
var (
	stepClasses     = []string{"BSC", "ISC"}
	setpointClasses = []string{"APC", "INC"}
)

// SetpointFormat is the point's format, or the protocol's default when it has none.
func SetpointFormat(protocol string, point *pyjson.Object) string {
	if format, _ := getOr(point, "format", "").(string); format != "" {
		return format
	}
	return SetpointFormats[protocol][0]
}

// typedReason is why a typed point cannot be served on this protocol ("" when it can), and
// it adds the format and word order a set point needs to the clean point.
func typedReason(protocol, kind string, raw, point *pyjson.Object) string {
	switch kind {
	case KindStep:
		if protocol == "modbus" {
			return "Modbus has no regulating step command; use IEC 104 or DNP3, or a set point"
		}
	case KindSetpoint:
		format := SetpointFormat(protocol, raw)
		if !contains(SetpointFormats[protocol], format) {
			return "the set point format must be one of " + strings.Join(SetpointFormats[protocol], ", ")
		}
		point.Set("format", format)
		if protocol == "modbus" {
			order, _ := getOr(raw, "word_order", WordOrders[0]).(string)
			if !contains(WordOrders, order) {
				return "the word order must be high_first (high word first) or low_first (low word first)"
			}
			point.Set("word_order", order)
		}
	}
	return ""
}

// controlClassReason is why a typed command cannot drive this control ("" when it can).
func controlClassReason(kind string, control *pyjson.Object, ref string) string {
	cdc, _ := getOr(control, "cdc", "").(string)
	allowed, name := stepClasses, "a regulating step (lower / raise)"
	if kind == KindSetpoint {
		allowed, name = setpointClasses, "a set point"
	}
	if contains(allowed, cdc) {
		return ""
	}
	if cdc == "" {
		cdc = "control of unknown class"
	}
	return ref + " is a " + cdc + " and cannot take " + name + " (it needs " + strings.Join(allowed, " or ") + ")"
}

// stepPositionReason is why a signal cannot feed a step position point ("" when it can).
func stepPositionReason(ref string, valueKind any) string {
	if text, _ := valueKind.(string); strings.HasPrefix(text, "INT") {
		return ""
	}
	return ref + " is not a whole number and cannot be sent as a step position (use valWTr.posVal)"
}
