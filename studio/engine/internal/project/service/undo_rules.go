package service

import (
	"studio-engine/internal/project/history"
	"studio-engine/internal/pyjson"
)

// undoRules tell the history what Undo leaves alone (ADR-0135):
//   - "id" (which project this is) and "network" (this PC's card, lab mode, GOOSE reach: the
//     lab service places real Windows addresses from it) are never part of a step;
//   - an IED's "lab_ip" (placed on the card by the network helper) and the gateway's
//     "auto_start" (it remembers whether it was running) belong to automatic changes, so
//     they are ignored when comparing and kept as they are now when a step is undone.
type undoRules struct{}

// compareDepth reaches an IED's fields from the project's IED list (list, IED, field) and
// one level more for the values that edits copy.
const compareDepth = 3

func (undoRules) Skip(key string) bool { return key == "id" || key == "network" }

func (undoRules) Same(key string, a, b any) bool {
	return history.Equal(a, b, compareDepth, automaticField(key))
}

func (undoRules) Keep(key string, restored, current any) any {
	switch key {
	case "ieds":
		return keepLabAddresses(restored, current)
	case "gateway":
		return keepField(restored, current, "auto_start")
	}
	return restored
}

func automaticField(key string) func(string) bool {
	switch key {
	case "ieds":
		return func(field string) bool { return field == "lab_ip" }
	case "gateway":
		return func(field string) bool { return field == "auto_start" }
	}
	return nil
}

// keepLabAddresses gives every restored IED that still exists its current lab address.
func keepLabAddresses(restored, current any) any {
	list, ok := restored.([]any)
	if !ok {
		return restored
	}
	now := map[any]*pyjson.Object{}
	if items, ok := current.([]any); ok {
		for _, item := range items {
			if ied, ok := item.(*pyjson.Object); ok {
				id, _ := ied.Get("id")
				now[id] = ied
			}
		}
	}
	out := make([]any, len(list))
	for index, item := range list {
		out[index] = item
		ied, ok := item.(*pyjson.Object)
		if !ok {
			continue
		}
		id, _ := ied.Get("id")
		if match, found := now[id]; found {
			out[index] = keepField(ied, match, "lab_ip")
		}
	}
	return out
}

// keepField returns restored with field set as in current (removed when current has none);
// restored itself is returned when they already agree, so unchanged parts keep their identity.
func keepField(restored, current any, field string) any {
	target, ok := restored.(*pyjson.Object)
	if !ok {
		return restored
	}
	var want any = history.Missing
	if source, ok := current.(*pyjson.Object); ok {
		if found, has := source.Get(field); has {
			want = found
		}
	}
	have, has := target.Get(field)
	if (has && history.Identical(have, want)) || (!has && want == history.Missing) {
		return target
	}
	out := shallow(target)
	if want == history.Missing {
		out.Delete(field)
	} else {
		out.Set(field, want)
	}
	return out
}

// stepLabel names a step in plain words for the Undo and Redo button tips.
func stepLabel(keys []string) string {
	has := map[string]bool{}
	for _, key := range keys {
		has[key] = true
	}
	for _, named := range []struct{ key, words string }{
		{"ieds", "IED change"}, {"diagram", "diagram change"}, {"gateway", "gateway change"},
		{"test_panel", "Test page change"}, {"sources", "test value change"}, {"scenarios", "sequence change"},
		{"snmp", "SNMP change"}, {"name", "project name change"},
	} {
		if has[named.key] {
			return named.words
		}
	}
	if len(keys) == 0 {
		return ""
	}
	return "project change"
}
