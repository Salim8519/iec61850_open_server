// Package testpanel is the saved Signal Test controls (projects/test_panel.py) and the
// panel view the Test place polls (api/test_panel.py).
package testpanel

import (
	"regexp"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
	"studio-engine/internal/tools/trace"
)

// MaxControls is the most saved controls a project keeps.
const MaxControls = 500

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}\z`)

func get(o *pyjson.Object, key string) any {
	if o == nil {
		return nil
	}
	value, _ := o.Get(key)
	return value
}

// Validate is validate_panel(entries, ieds): the clean list to store, or Python's error.
func Validate(entries any, ieds []*pyjson.Object) ([]any, error) {
	list, ok := entries.([]any)
	if !ok || len(list) > MaxControls {
		return nil, apperr.New("Signal Test supports at most 500 saved controls")
	}
	models := map[string]*pyjson.Object{}
	for _, ied := range ieds {
		if id, isText := get(ied, "id").(string); isText {
			models[id] = ied
		}
	}
	result, ids := []any{}, map[string]bool{}
	for _, raw := range list {
		entry, isObject := raw.(*pyjson.Object)
		if !isObject {
			return nil, apperr.New("Invalid saved signal control")
		}
		key, keyText := get(entry, "id").(string)
		if !keyText || !idPattern.MatchString(key) || ids[key] {
			return nil, apperr.New("Saved control IDs must be valid and unique")
		}
		label, labelText := get(entry, "label").(string)
		if !labelText || pytext.Len(pytext.Strip(label)) < 1 || pytext.Len(pytext.Strip(label)) > 100 {
			return nil, apperr.New("Saved control label must contain 1 to 100 characters")
		}
		iedID, iedText := get(entry, "ied_id").(string)
		ref, refText := get(entry, "ref").(string)
		if !iedText || !refText {
			return nil, apperr.New("Saved control needs an IED and exact signal reference")
		}
		ied := models[iedID]
		signal := findSignal(ied, ref)
		if signal == nil || !pytext.Truthy(get(signal, "writable")) || get(signal, "fc") == "CF" {
			return nil, apperr.New("Choose a supported runtime-editable signal")
		}
		if _, err := trace.ValidateValue(signal, get(signal, "value")); err != nil {
			return nil, err
		}
		reportRef, _ := entry.Get("report_ref")
		if err := ValidateMapping(ied, get(signal, "ref"), reportRef); err != nil {
			return nil, err
		}
		duration, found := entry.Get("duration_ms")
		if !found {
			duration = int64(1000)
		}
		active, found := entry.Get("active_value")
		if !found {
			active = true
		}
		if !pytext.IntIn(duration, 100, 60000) {
			return nil, apperr.New("Pulse duration must be 100 to 60000 milliseconds")
		}
		if _, isBool := active.(bool); !isBool {
			return nil, apperr.New("Pulse active value must be Boolean")
		}
		if !pytext.Truthy(reportRef) {
			reportRef = nil
		}
		result = append(result, pyjson.NewObject().Set("id", key).Set("label", pytext.Strip(label)).
			Set("ied_id", get(ied, "id")).Set("ref", get(signal, "ref")).Set("report_ref", reportRef).
			Set("duration_ms", duration).Set("active_value", active))
		ids[key] = true
	}
	return result, nil
}

func findSignal(ied *pyjson.Object, ref string) *pyjson.Object {
	signals, _ := get(ied, "signals").([]any)
	for _, raw := range signals {
		if signal, ok := raw.(*pyjson.Object); ok && get(signal, "ref") == ref {
			return signal
		}
	}
	return nil
}

// ValidateMapping is validate_mapping: a report reference is text or null, and a chosen
// report must carry the signal in its dataset.
func ValidateMapping(ied *pyjson.Object, reference any, reportRef any) error {
	_, isText := reportRef.(string)
	if reportRef != nil && !isText {
		return apperr.New("Report reference must be text or null")
	}
	if !pytext.Truthy(reportRef) {
		return nil
	}
	members, err := ReportMembers(ied, reportRef)
	if err != nil {
		return err
	}
	if !members[pytext.Str(reference)] {
		return apperr.New("Selected signal is not a member of the chosen report")
	}
	return nil
}

// ReportMembers is scl/report_catalogue.report_members: the signal refs of the report's dataset.
func ReportMembers(ied *pyjson.Object, reference any) (map[string]bool, error) {
	var report *pyjson.Object
	for _, raw := range asList(get(ied, "reports")) {
		if item, ok := raw.(*pyjson.Object); ok && pytext.Equal(get(item, "ref"), reference) {
			report = item
			break
		}
	}
	if report == nil {
		return nil, apperr.New("Unknown report reference")
	}
	members := map[string]bool{}
	for _, raw := range asList(get(ied, "datasets")) {
		item, ok := raw.(*pyjson.Object)
		if ok && pytext.Equal(get(item, "ref"), get(report, "dataset_ref")) {
			for _, ref := range asList(get(item, "signal_refs")) {
				members[pytext.Str(ref)] = true
			}
			break
		}
	}
	return members, nil
}

func asList(value any) []any {
	items, _ := value.([]any)
	return items
}
