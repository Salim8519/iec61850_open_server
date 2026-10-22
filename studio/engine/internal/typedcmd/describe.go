package typedcmd

import (
	"strconv"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

func text(o *pyjson.Object, key string) string {
	value, _ := o.Get(key)
	s, _ := value.(string)
	return s
}

func whole(o *pyjson.Object, key string) int64 {
	value, _ := o.Get(key)
	switch v := value.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	}
	return 0
}

// FromStandard describes an SPC/DPC entry of an IED's "controls" (on/off values).
func FromStandard(entry *pyjson.Object) Control {
	return Control{Ref: text(entry, "ref"), CDC: text(entry, "cdc"), ValueKind: "bool",
		Model: whole(entry, "model"), SboMs: whole(entry, "sbo_timeout_ms")}
}

// FromExtended describes one scl.ExtendedControls entry.
func FromExtended(entry *pyjson.Object) Control {
	labels, _ := entry.Get("labels")
	object, _ := labels.(*pyjson.Object)
	low, _ := entry.Get("min")
	high, _ := entry.Get("max")
	return Control{Ref: text(entry, "ref"), CDC: text(entry, "cdc"), ValueKind: text(entry, "value_kind"),
		Labels: object, Min: low, Max: high, Model: whole(entry, "model"), SboMs: whole(entry, "sbo_timeout_ms")}
}

// Outcome turns a native typed reply into an error when the device refused: plain English,
// HTTP 409 (the request was fine; the device said no).
func Outcome(result *pyjson.Object) error {
	if ok, _ := result.Get("ok"); ok == true {
		return nil
	}
	message := text(result, "error")
	if message == "" {
		message = "The device refused the command"
	}
	return apperr.Coded(message, "conflict")
}

// SelectExpired is the text for an Operate that came after the select timed out.
func SelectExpired(timeoutMs int64) error {
	seconds := (timeoutMs + 999) / 1000
	return apperr.Coded("Select expired after "+strconv.FormatInt(seconds, 10)+" s. Press Select again.", "conflict")
}
