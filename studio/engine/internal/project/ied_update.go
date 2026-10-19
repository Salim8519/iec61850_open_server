package project

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project/netset"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

var editableSettings = []string{"role", "live", "lab_ip", "goose_publish", "port"}

// UpdateIED is ied_settings.update_ied without the save: the IED's role, live, lab_ip,
// goose_publish and port are checked, stored on the IED inside project, and returned.
func UpdateIED(project *pyjson.Object, iedID string, changes any) (*pyjson.Object, error) {
	given, isDict := changes.(*pyjson.Object)
	if !isDict || given.Len() == 0 || !onlyEditable(given) {
		return nil, apperr.New("Only role, live, lab_ip, goose_publish and port can be changed here")
	}
	ied, err := FindIED(project, iedID)
	if err != nil {
		return nil, err
	}
	merged := pyjson.NewObject()
	for _, key := range []string{"role", "live", "lab_ip", "goose_publish", "test_mode"} {
		merged.Set(key, pyval.Lookup(ied, key, nil))
	}
	if pyval.Has(given, "live") {
		live, err := pyval.Merge(pyval.Or(pyval.Lookup(ied, "live", nil), pyjson.NewObject()),
			pyval.Or(pyval.Lookup(given, "live", nil), pyjson.NewObject()))
		if err != nil {
			return nil, err
		}
		given = pyval.CopyShallow(given).Set("live", live)
	}
	for _, key := range given.Keys() {
		if key != "port" {
			value, _ := given.Get(key)
			merged.Set(key, value)
		}
	}
	name := pyval.Str(pyval.Lookup(ied, "name", nil))
	settings, err := netset.ValidateIEDSettings(merged)
	if err != nil {
		if message, ok := asValueError(err); ok {
			return nil, apperr.New(name + ": " + message)
		}
		return nil, err
	}
	port := pyval.Lookup(given, "port", pyval.Lookup(ied, "port", int64(102)))
	if !pyval.IsInt(port) || !pyval.Between(port, 1, 65535) {
		return nil, apperr.New(name + ": Port must be a whole number from 1 to 65535 (MMS normally uses 102)")
	}
	trial := []*pyjson.Object{}
	for _, item := range projectIEDs(project) {
		if item == ied {
			candidate := appendAll(pyval.CopyShallow(item), settings)
			item = candidate.Set("port", port)
		}
		trial = append(trial, item)
	}
	problems, err := EndpointConflicts(trial, object(pyval.Lookup(project, "network", nil)))
	if err != nil {
		return nil, err
	}
	if len(problems) > 0 {
		return nil, conflictError(problems)
	}
	appendAll(ied, settings).Set("port", port)
	result := pyjson.NewObject()
	for _, key := range editableSettings {
		value, _ := ied.Get(key)
		result.Set(key, value)
	}
	return result, nil
}

func onlyEditable(changes *pyjson.Object) bool {
	for _, key := range changes.Keys() {
		if !pyval.InTuple(key, editableSettings...) {
			return false
		}
	}
	return true
}
