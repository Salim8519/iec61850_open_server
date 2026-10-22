package runtime

import (
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/scl"
)

// Control entries of an IED (ADR-0108). `controls` of the IED payload holds the SPC and DPC
// objects the canvas has always operated; the typed list of ADR-0104 (INC, ENC, APC, BAC, BSC,
// ISC) is never part of the payload: scl.ExtendedControls reads it from the stored SCL (the same
// list GET /controls/typed answers as `extended_controls`). Both carry the numeric ctlModel.
// The first list wins, so an SPC or DPC keeps the entry it has today.

// ctlModelNames are the IEC 61850-7-2 ctlModel values 0 to 4 as the typed routes name them.
var ctlModelNames = []string{"status-only", "direct-normal", "sbo-normal", "direct-enhanced", "sbo-enhanced"}

// typedCDCs are the control classes a typed command may address.
var typedCDCs = map[string]bool{"SPC": true, "DPC": true, "INC": true, "ENC": true, "APC": true, "BAC": true,
	"BSC": true, "ISC": true}

// extendedControlsOf is the IED's typed control list (ADR-0104 `extended_controls`), nil when its
// SCL cannot be read. A variable only so tests can give an IED a list without writing SCL.
var extendedControlsOf = func(ied *pyjson.Object) []any {
	list, err := scl.CachedExtendedControls(pyval.Str(ied, "id"), pyval.Str(ied, "name"), pyval.Str(ied, "source"))
	if err != nil {
		return nil
	}
	return list
}

// typedControl finds a control by reference in `controls`, then in the typed list.
func typedControl(ied *pyjson.Object, ref string) *pyjson.Object {
	if found := controlByRef(pyval.List(ied, "controls"), ref); found != nil {
		return found
	}
	return controlByRef(extendedControlsOf(ied), ref)
}

func controlByRef(list []any, ref string) *pyjson.Object {
	for _, item := range list {
		if entry, ok := item.(*pyjson.Object); ok && pyval.Str(entry, "ref") == ref {
			return entry
		}
	}
	return nil
}

// controlModelName is the name of the control's numeric ctlModel; "" when it is not 0 to 4.
func controlModelName(control *pyjson.Object) string {
	switch model := pyval.Get(control, "model").(type) {
	case int64:
		if model >= 0 && int(model) < len(ctlModelNames) {
			return ctlModelNames[model]
		}
	case float64:
		if model >= 0 && int(model) < len(ctlModelNames) && model == float64(int(model)) {
			return ctlModelNames[int(model)]
		}
	}
	return ""
}

// isSboModel says whether a command to this model needs Select before Operate.
func isSboModel(model string) bool { return strings.HasPrefix(model, "sbo-") }

// operateWindow is how long the browser offers Operate after a Select: the selection's own time
// minus a safety margin for the round trip (3 s of a long timeout, a tenth of a short one), so
// the IED never sees an Operate after its own timer ran out. 30 s gives 27 s.
func operateWindow(expiresMs int) int {
	if expiresMs <= 0 {
		return 0
	}
	if expiresMs > 6000 {
		return expiresMs - 3000
	}
	return expiresMs * 9 / 10
}
