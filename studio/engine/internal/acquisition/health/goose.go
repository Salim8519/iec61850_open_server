package health

import (
	"fmt"
	"strings"

	"studio-engine/internal/acquisition/modes"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// problemText is PROBLEM_TEXT: how a GOOSE watch problem reads in a sentence.
func problemText(problem string) (string, bool) {
	switch problem {
	case "conf_rev_mismatch":
		return "arrives with a different configuration revision than the SCD", true
	case "dataset_mismatch":
		return "arrives with a different dataset than the SCD", true
	case "go_id_mismatch":
		return "arrives with a different GOOSE ID than the SCD", true
	case "invalid_frame":
		return "arrives but cannot be decoded", true
	case "needs_commission":
		return "says its publisher still needs commissioning", true
	case "timeout":
		return "stopped arriving within its allowed time", true
	}
	return "", false
}

// sameReference is _same_reference: equal once "." and "$" separators are unified.
func sameReference(a, b any) bool {
	text := func(value any) string {
		if !pyv.Truthy(value) {
			return ""
		}
		return strings.ReplaceAll(pyv.PyStr(value), ".", "$")
	}
	return text(a) == text(b)
}

// DeviceGocbChecks is device_gocb_checks: GoID, DatSet and ConfRev read from the device
// (reads = {values, checked_at}) compared with each SCD GOOSE control block.
func DeviceGocbChecks(ied, reads *pyjson.Object) []any {
	rows := []any{}
	values := pyv.GetObj(reads, "values")
	when := pyv.Get(reads, "checked_at")
	for _, control := range pyv.Objects(pyv.GetList(pyv.OrObj(ied, "goose"), "publishes")) {
		base := modes.GocbObject(pyv.GetStr(control, "gocb_ref"))
		got := pyjson.NewObject()
		allNone := true
		for _, name := range modes.GocbAttributes() {
			value := pyv.Get(values, base+"."+name)
			got.Set(name, value)
			allNone = allNone && value == nil
		}
		item := pyv.Get(control, "gocb_ref")
		if allNone {
			rows = append(rows, row("goose_device", item, "warning", "The device did not return this GOOSE block. "+
				"It may run a different configuration than the SCD.", when, "mms_read", nil))
			continue
		}
		var differences []string
		if goID := pyv.Get(got, "GoID"); goID != nil && !pyv.Equal(goID, pyv.Get(control, "go_id")) {
			differences = append(differences, fmt.Sprintf("GOOSE ID is %s on the device but %s in the SCD",
				pyv.PyStr(goID), pyv.PyStr(pyv.Get(control, "go_id"))))
		}
		if dataset := pyv.Get(got, "DatSet"); dataset != nil && !sameReference(dataset, pyv.Get(control, "dataset_ref")) {
			differences = append(differences, fmt.Sprintf("dataset is %s on the device but %s in the SCD",
				pyv.PyStr(dataset), pyv.PyStr(pyv.Get(control, "dataset_ref"))))
		}
		if confRev := pyv.Get(got, "ConfRev"); confRev != nil && !pyv.Equal(confRev, pyv.Get(control, "conf_rev")) {
			differences = append(differences, fmt.Sprintf("confRev is %s on the device but %s in the SCD",
				pyv.PyStr(confRev), pyv.PyStr(pyv.Get(control, "conf_rev"))))
		}
		enabled, isBool := pyv.Get(got, "GoEna").(bool)
		switch {
		case len(differences) > 0:
			rows = append(rows, row("goose_device", item, "problem", Sentence(differences), when, "mms_read", got))
		case isBool && !enabled:
			rows = append(rows, row("goose_device", item, "warning", "This GOOSE block is switched off on the device.",
				when, "mms_read", got))
		default:
			rows = append(rows, row("goose_device", item, "ok", "Matches the SCD.", when, "mms_read", got))
		}
	}
	return rows
}

// WireCheck is wire_check: the verdict for one GOOSE block from what the passive listener
// saw (listener is the listener snapshot with its state and message).
func WireCheck(watch, listener *pyjson.Object) *pyjson.Object {
	item := pyv.Get(watch, "gocb_ref")
	if !pyv.Equal(pyv.Get(listener, "state"), "listening") {
		message := pyv.Get(listener, "message")
		if !pyv.Truthy(message) {
			message = "The GOOSE listener is not running."
		}
		return notChecked("goose_wire", item, message)
	}
	when, state, problem := pyv.Get(watch, "observed_at"), pyv.Get(watch, "state"), pyv.Get(watch, "problem")
	if pyv.Equal(state, "receiving") {
		if pyv.Truthy(pyv.Get(watch, "received_test")) {
			return row("goose_wire", item, "warning", "Arriving, marked as test/simulation traffic.", when, "wire", nil)
		}
		message := fmt.Sprintf("Arriving as engineered (%s messages).", pyv.PyStr(pyv.GetOr(watch, "frames", int64(0))))
		return row("goose_wire", item, "ok", message, when, "wire", nil)
	}
	if pyv.Equal(state, "problem") || pyv.Equal(state, "timeout") {
		detail := pyv.O("conf_rev", pyv.Get(watch, "received_conf_rev"), "go_id", pyv.Get(watch, "received_go_id"),
			"dataset", pyv.Get(watch, "received_dataset"))
		expected := pyv.O("conf_rev", pyv.Get(watch, "conf_rev"), "go_id", pyv.Get(watch, "go_id"),
			"dataset", pyv.Get(watch, "dataset_ref"))
		key := problem
		if !pyv.Truthy(key) {
			key = state
		}
		text, known := problemText(pyv.Str(key))
		if _, isText := key.(string); !isText || !known {
			text = "arrives with a problem"
		}
		if named := namedField(problem); named != "" {
			text += fmt.Sprintf(" (wire %s, SCD %s)", pyv.PyStr(pyv.Get(detail, named)), pyv.PyStr(pyv.Get(expected, named)))
		}
		return row("goose_wire", item, "problem", "This GOOSE "+text+".", when, "wire", detail)
	}
	return row("goose_wire", item, "warning", "Not seen on the network yet.", when, "wire", nil)
}

// namedField is the detail field a mismatch problem names, or "".
func namedField(problem any) string {
	switch pyv.Str(problem) {
	case "conf_rev_mismatch":
		return "conf_rev"
	case "dataset_mismatch":
		return "dataset"
	case "go_id_mismatch":
		return "go_id"
	}
	return ""
}

// UnknownPublishers is unknown_publishers: copies of the GOOSE blocks on the wire that the
// project does not describe, plus a note when the listing limit was reached.
func UnknownPublishers(status *pyjson.Object) ([]any, string) {
	rows := []any{}
	for _, item := range pyv.Objects(pyv.List(pyv.GetOr(status, "unknown", []any{}))) {
		rows = append(rows, pyv.Shallow(item))
	}
	limit := pyv.Get(status, "unknown_limit")
	if !pyv.Truthy(limit) {
		return rows, ""
	}
	text := pyv.PyStr(limit)
	return rows, fmt.Sprintf("More than %s unknown GOOSE blocks were seen; only the first %s are listed.", text, text)
}
