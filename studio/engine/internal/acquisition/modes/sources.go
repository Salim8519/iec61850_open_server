package modes

import (
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// SimulationSources is simulation_sources: every observed source of simulation- or
// test-flagged GOOSE or Sampled Values traffic, from the listener snapshots.
func SimulationSources(goose, sv *pyjson.Object) []any {
	sources := []any{}
	for _, watch := range pyv.Objects(pyv.List(pyv.GetOr(goose, "watches", []any{}))) {
		state := pyv.Get(watch, "state")
		if pyv.Truthy(pyv.Get(watch, "received_test")) && (pyv.Equal(state, "receiving") || pyv.Equal(state, "problem")) {
			name := pyv.Get(pyv.OrObj(watch, "publisher"), "name")
			if !pyv.Truthy(name) {
				name = "unknown"
			}
			sources = append(sources, "GOOSE "+pyv.PyStr(pyv.Get(watch, "gocb_ref"))+" from "+pyv.PyStr(name))
		}
	}
	for _, item := range pyv.Objects(pyv.List(pyv.GetOr(goose, "unknown", []any{}))) {
		if pyv.Truthy(pyv.Get(item, "test")) {
			sources = append(sources, "GOOSE "+pyv.PyStr(pyv.Get(item, "gocb_ref"))+" (not in project)")
		}
	}
	for _, watch := range pyv.Objects(pyv.List(pyv.GetOr(sv, "watches", []any{}))) {
		if contains(pyv.GetOr(watch, "problems", []any{}), "simulation") {
			name := pyv.Get(watch, "sv_id")
			if !pyv.Truthy(name) {
				name = pyv.Get(watch, "key")
			}
			sources = append(sources, "Sampled Values "+pyv.PyStr(name)+" from "+pyv.PyStr(pyv.Get(watch, "publisher")))
		}
	}
	for _, item := range pyv.Objects(pyv.List(pyv.GetOr(sv, "unknown", []any{}))) {
		if pyv.Truthy(pyv.Get(item, "simulate")) {
			name := pyv.Get(item, "sv_id")
			if !pyv.Truthy(name) {
				name = pyv.Get(item, "app_id")
			}
			sources = append(sources, "Sampled Values "+pyv.PyStr(name)+" (not in project)")
		}
	}
	return sources
}

// contains is Python's `item in container` for a list (equality), a string (substring) or
// an object (key).
func contains(container any, item string) bool {
	switch v := container.(type) {
	case []any:
		for _, element := range v {
			if pyv.Equal(element, item) {
				return true
			}
		}
	case string:
		return strings.Contains(v, item)
	case *pyjson.Object:
		return pyv.Has(v, item)
	}
	return false
}
