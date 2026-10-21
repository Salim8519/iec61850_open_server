package project

import (
	"regexp"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/value"
)

// identifierPattern is re.fullmatch(r"[A-Za-z0-9_-]{1,64}", text).
var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func validIdentifier(v any) bool {
	text, ok := v.(string)
	return ok && identifierPattern.MatchString(text)
}

// reportMembers is report_catalogue.report_members: the signal references of the dataset
// behind a report.
func reportMembers(ied *pyjson.Object, reference string) (map[string]bool, error) {
	report := findByRef(pyval.Lookup(ied, "reports", nil), reference)
	if report == nil {
		return nil, apperr.New("Unknown report reference")
	}
	members := map[string]bool{}
	datasetRef := pyval.Lookup(report, "dataset_ref", nil)
	for _, item := range items(pyval.Lookup(ied, "datasets", nil)) {
		if dataset := object(item); pyval.Equal(pyval.Lookup(dataset, "ref", nil), datasetRef) {
			for _, ref := range items(pyval.Lookup(dataset, "signal_refs", nil)) {
				members[pyval.Str(ref)] = true
			}
			break
		}
	}
	return members, nil
}

// validateMapping is test_panel.validate_mapping: an optional report must contain the signal.
func validateMapping(ied *pyjson.Object, reference string, reportRef any) error {
	report, isText := reportRef.(string)
	if reportRef != nil && !isText {
		return apperr.New("Report reference must be text or null")
	}
	if report == "" {
		return nil
	}
	members, err := reportMembers(ied, report)
	if err != nil {
		return err
	}
	if !members[reference] {
		return apperr.New("Selected signal is not a member of the chosen report")
	}
	return nil
}

// ValidatePanel is validate_panel: the saved Signal Test controls.
func ValidatePanel(entries any, ieds []*pyjson.Object) ([]any, error) {
	list, isList := entries.([]any)
	if !isList || len(list) > 500 {
		return nil, apperr.New("Signal Test supports at most 500 saved controls")
	}
	models := map[string]*pyjson.Object{}
	for _, ied := range ieds {
		models[text(ied, "id")] = ied
	}
	result, ids := []any{}, map[string]bool{}
	for _, item := range list {
		entry, isDict := item.(*pyjson.Object)
		if !isDict {
			return nil, apperr.New("Invalid saved signal control")
		}
		key, label := pyval.Lookup(entry, "id", nil), pyval.Lookup(entry, "label", nil)
		if !validIdentifier(key) || ids[key.(string)] {
			return nil, apperr.New("Saved control IDs must be valid and unique")
		}
		labelText, isText := label.(string)
		if length := pyval.Len(pyrules.Strip(labelText)); !isText || length < 1 || length > 100 {
			return nil, apperr.New("Saved control label must contain 1 to 100 characters")
		}
		iedID, idIsText := pyval.Lookup(entry, "ied_id", nil).(string)
		ref, refIsText := pyval.Lookup(entry, "ref", nil).(string)
		if !idIsText || !refIsText {
			return nil, apperr.New("Saved control needs an IED and exact signal reference")
		}
		ied := models[iedID]
		var signal *pyjson.Object
		if ied != nil {
			signal = findByRef(pyval.Lookup(ied, "signals", nil), ref)
		}
		if signal == nil || !pyval.Truthy(pyval.Lookup(signal, "writable", nil)) || text(signal, "fc") == "CF" {
			return nil, apperr.New("Choose a supported runtime-editable signal")
		}
		if _, err := value.Validate(signal, pyval.Lookup(signal, "value", nil)); err != nil {
			return nil, err
		}
		reportRef := pyval.Lookup(entry, "report_ref", nil)
		if err := validateMapping(ied, text(signal, "ref"), reportRef); err != nil {
			return nil, err
		}
		duration := pyval.Lookup(entry, "duration_ms", int64(1000))
		active := pyval.Lookup(entry, "active_value", true)
		if !pyval.IsInt(duration) || !pyval.Between(duration, 100, 60000) {
			return nil, apperr.New("Pulse duration must be 100 to 60000 milliseconds")
		}
		if _, ok := active.(bool); !ok {
			return nil, apperr.New("Pulse active value must be Boolean")
		}
		result = append(result, pyjson.NewObject().Set("id", key).Set("label", pyrules.Strip(labelText)).
			Set("ied_id", text(ied, "id")).Set("ref", text(signal, "ref")).
			Set("report_ref", pyval.Or(reportRef, nil)).Set("duration_ms", duration).Set("active_value", active))
		ids[key.(string)] = true
	}
	return result, nil
}
