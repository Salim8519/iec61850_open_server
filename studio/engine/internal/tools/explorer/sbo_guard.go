package explorer

import (
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// SBONotRead is the value text of a control's SBO attribute, which Explorer never reads (change 0177).
// In IEC 61850-8-1 a Select of a "select before operate, normal security" control IS an MMS read of
// <DO>.SBO: the device reserves the control for this client. Reading it to show a value would select
// the switch behind the user's back, and polling it would keep it selected.
const SBONotRead = "Not read: reading SBO would select the control"

// SelectsOnRead reports whether reading this attribute performs a Select: functional constraint CO and
// the last name element SBO (dot or MMS "$" form). SBOw, Oper and Cancel reads have no side effect.
func SelectsOnRead(ref, fc any) bool {
	if !strings.EqualFold(pytext.Strip(pytext.Str(fc)), "CO") {
		return false
	}
	name := pytext.Str(ref)
	if cut := strings.LastIndexAny(name, ".$"); cut >= 0 {
		name = name[cut+1:]
	}
	return strings.EqualFold(name, "SBO")
}

// notRead is the value entry shown instead of reading an SBO attribute; it is never an error.
func notRead(now float64) *pyjson.Object {
	return pyjson.NewObject().Set("raw", nil).Set("text", SBONotRead).Set("status", "warning").Set("at", now)
}

// splitSBO separates the attributes Explorer may read from the SBO ones it answers without reading.
func splitSBO(items []*pyjson.Object) (readable []*pyjson.Object, skipped []string) {
	for _, item := range items {
		if SelectsOnRead(get(item, "ref"), get(item, "fc")) {
			skipped = append(skipped, pytext.Str(get(item, "ref")))
		} else {
			readable = append(readable, item)
		}
	}
	return readable, skipped
}
