package service

import "studio-engine/internal/pyjson"

// Copy-on-write helpers: a mutation works on a shallow copy of the published project (and
// of the IED objects it may touch), so readers of the previous state never see a change.

func shallow(o *pyjson.Object) *pyjson.Object {
	out := pyjson.NewObject()
	for _, key := range o.Keys() {
		value, _ := o.Get(key)
		out.Set(key, value)
	}
	return out
}

// draft returns a copy of the project whose "ieds" list and IED objects are fresh copies.
func draft(project *pyjson.Object) *pyjson.Object {
	next := shallow(project)
	raw, _ := project.Get("ieds")
	list, _ := raw.([]any)
	ieds := make([]any, len(list))
	for i, item := range list {
		if ied, ok := item.(*pyjson.Object); ok {
			ieds[i] = shallow(ied)
		} else {
			ieds[i] = item
		}
	}
	next.Set("ieds", ieds)
	return next
}

// iedList returns the project's IEDs as objects.
func iedList(project *pyjson.Object) []*pyjson.Object {
	raw, _ := project.Get("ieds")
	list, _ := raw.([]any)
	out := make([]*pyjson.Object, 0, len(list))
	for _, item := range list {
		if ied, ok := item.(*pyjson.Object); ok {
			out = append(out, ied)
		}
	}
	return out
}

func objects(items []*pyjson.Object) []any {
	out := make([]any, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out
}
