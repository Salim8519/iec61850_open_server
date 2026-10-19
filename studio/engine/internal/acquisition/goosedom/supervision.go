package goosedom

import (
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// lgosPoints is LGOS_POINTS in Python order: LGOS attribute suffix and received field.
var lgosPoints = [4][2]string{{"St.stVal", "receiving"}, {"SimSt.stVal", "test"},
	{"LastStNum.stVal", "st_num"}, {"ConfRevNum.stVal", "conf_rev"}}

// LgosPoints returns LGOS_POINTS as a fresh object (suffix -> received field).
func LgosPoints() *pyjson.Object {
	result := pyjson.NewObject()
	for _, point := range lgosPoints {
		result.Set(point[0], point[1])
	}
	return result
}

// Update is one attribute write: the signal reference and its new value.
type Update struct {
	Ref   string
	Value any
}

// WatchList returns every GoCB to observe, keyed by reference in Python insertion
// order, with who publishes and subscribes. The returned watches are new objects.
func WatchList(ieds []*pyjson.Object) *pyjson.Object {
	watches := pyjson.NewObject()
	for _, ied := range ieds {
		role := pyv.GetOr(ied, "role", "simulated")
		if pyv.Equal(role, "off") {
			continue
		}
		goose := pyv.OrObj(ied, "goose")
		for _, control := range objects(goose, "publishes") {
			entry := watchEntry(watches, control)
			entry.Set("publisher", pyv.O("ied_id", pyv.Get(ied, "id"), "name", pyv.Get(ied, "name"), "role", role))
		}
		if !pyv.Equal(role, "simulated") {
			continue
		}
		for _, subscription := range objects(goose, "subscribes") {
			entry := watchEntry(watches, subscription)
			if !pyv.Has(entry, "publisher") {
				entry.Set("publisher", pyv.O("ied_id", nil, "name", pyv.GetOr(subscription, "publisher", ""),
					"role", "outside_project"))
			}
			inputs := pyv.List(pyv.GetOr(subscription, "inputs", []any{}))
			subscribers := append(pyv.GetList(entry, "subscribers"), pyv.O("ied_id", pyv.Get(ied, "id"),
				"name", pyv.Get(ied, "name"), "lgos", pyv.Get(subscription, "lgos"), "inputs", int64(len(inputs))))
			entry.Set("subscribers", subscribers)
		}
	}
	return watches
}

// watchEntry is watches.setdefault(control["gocb_ref"], _entry(control)).
func watchEntry(watches, control *pyjson.Object) *pyjson.Object {
	key := pyv.PyStr(pyv.Get(control, "gocb_ref"))
	if existing, found := watches.Get(key); found {
		return existing.(*pyjson.Object)
	}
	entry := pyv.O("key", pyv.Get(control, "gocb_ref"), "gocb_ref", pyv.Get(control, "gocb_ref"),
		"app_id", pyv.Get(control, "app_id"), "mac", pyv.Get(control, "mac"), "go_id", pyv.Get(control, "go_id"),
		"dataset_ref", pyv.Get(control, "dataset_ref"), "conf_rev", pyv.Get(control, "conf_rev"),
		"subscribers", []any{})
	watches.Set(key, entry)
	return entry
}

// LgosUpdates returns the LGOS attribute writes for one subscriber; only LGOS points
// present in index (signal reference -> signal) are touched.
func LgosUpdates(lgos any, received *pyjson.Object, index map[string]*pyjson.Object) []Update {
	if !pyv.Truthy(lgos) {
		return nil
	}
	values := map[string]any{
		"receiving": pyv.Equal(pyv.Get(received, "state"), "receiving"),
		"test":      pyv.Truthy(pyv.Get(received, "test")),
		"st_num":    pyv.Get(received, "st_num"),
		"conf_rev":  pyv.Get(received, "conf_rev"),
	}
	var updates []Update
	for _, point := range lgosPoints {
		reference := pyv.PyStr(lgos) + "." + point[0]
		signal := index[reference]
		value := values[point[1]]
		if signal == nil || value == nil {
			continue
		}
		if pyv.Equal(pyv.Get(signal, "kind"), "BOOLEAN") {
			value = pyv.Truthy(value)
		} else if !pyv.IsInt(value) {
			continue
		}
		updates = append(updates, Update{Ref: reference, Value: value})
	}
	return updates
}
