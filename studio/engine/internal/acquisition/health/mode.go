package health

import (
	"fmt"
	"strings"

	"studio-engine/internal/acquisition/modes"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// ModeRows is mode_rows: one row per Mod/Beh/Sim point the IED model has, from the mode
// read (mode = {values, errors, checked_at}).
func ModeRows(ied, mode *pyjson.Object) []any {
	items := modes.ModeItems(ied)
	if len(items) == 0 {
		return []any{notChecked("mode", "Mode", "Mode, behaviour and simulation are not available in this IED.")}
	}
	rows := []any{}
	when := pyv.Get(mode, "checked_at")
	failed := pyv.Get(mode, "errors")
	values := pyv.GetObj(mode, "values")
	for _, item := range pyv.Objects(items) {
		ref := pyv.GetStr(item, "ref")
		if containsKey(failed, ref) {
			rows = append(rows, row("mode", ref, "warning", "Could not be read from the device.", when, "mms_read", nil))
			continue
		}
		value := pyv.Get(values, ref)
		var verdict, message string
		if strings.HasSuffix(ref, ".Sim.stVal") {
			verdict, message = "ok", "Not in simulation mode."
			if on, isBool := value.(bool); isBool && on {
				verdict = "warning"
				message = "The IED is in simulation mode: it listens to test GOOSE/SV instead of the real ones."
			}
		} else {
			name, known := modes.ModeName(value)
			verdict = "warning"
			if name == "on" {
				verdict = "ok"
			}
			switch {
			case !known:
				message = fmt.Sprintf("The device reports %s, which is not a valid mode (1 to 5).", pyv.PyStr(value))
			case name != "on":
				message = fmt.Sprintf("This logical device is %s, not in normal service.", name)
			default:
				message = upperFirst(name) + "."
			}
		}
		rows = append(rows, row("mode", ref, verdict, message, when, "mms_read", pyv.O("value", value)))
	}
	return rows
}

// containsKey is Python's `key in container` for an object (key), a list (equality) or a
// string (substring).
func containsKey(container any, key string) bool {
	switch v := container.(type) {
	case *pyjson.Object:
		return pyv.Has(v, key)
	case []any:
		for _, element := range v {
			if pyv.Equal(element, key) {
				return true
			}
		}
	case string:
		return strings.Contains(v, key)
	}
	return false
}
