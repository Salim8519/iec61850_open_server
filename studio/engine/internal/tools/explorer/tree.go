// Package explorer is IED Explorer: the node view, value texts and key-point watch rules
// (domain/explorer_*.py) and the service that browses, polls and operates (tools/explorer*.py).
package explorer

import (
	"sort"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// blockFCs are control-block service attributes, shown as blocks rather than data objects.
var blockFCs = map[string]bool{"RP": true, "BR": true, "GO": true, "GS": true, "MS": true, "US": true, "LG": true, "SV": true}

// fcOrder: live status and measurements first, settings and description next, controls last.
var fcOrder = map[string]int{}

func init() {
	for rank, fc := range []string{"ST", "MX", "SP", "SG", "SE", "SV", "CF", "DC", "EX", "BL", "OR", "CO"} {
		fcOrder[fc] = rank
	}
}

func rank(fc any) int {
	if name, ok := fc.(string); ok {
		if value, found := fcOrder[name]; found {
			return value
		}
	}
	return len(fcOrder)
}

func get(o *pyjson.Object, key string) any {
	if o == nil {
		return nil
	}
	value, _ := o.Get(key)
	return value
}

func list(value any) []any {
	items, _ := value.([]any)
	return items
}

// tail is text[from:] counted in code points.
func tail(text string, from int) string {
	return strings.TrimPrefix(text, pytext.Head(text, from))
}

// NodeView is node_view(reference, answer) for the native "explore node" answer.
func NodeView(reference string, answer *pyjson.Object) *pyjson.Object {
	objects := map[string]*pyjson.Object{}
	var names []string
	cut := pytext.Len(reference) + 1
	for _, raw := range list(get(answer, "leaves")) {
		leaf, _ := raw.(*pyjson.Object)
		fc := pytext.Str(get(leaf, "fc"))
		if blockFCs[fc] {
			continue
		}
		rest := tail(pytext.Str(get(leaf, "ref")), cut)
		name, _, _ := strings.Cut(rest, ".")
		entry, found := objects[name]
		if !found {
			entry = pyjson.NewObject().Set("name", name).Set("fcs", []any{}).Set("attributes", []any{})
			objects[name] = entry
			names = append(names, name)
		}
		fcs := list(get(entry, "fcs"))
		if !contains(fcs, fc) {
			entry.Set("fcs", append(fcs, fc))
		}
		path := tail(rest, pytext.Len(name)+1)
		if path == "" {
			path = name
		}
		attribute := pyjson.NewObject().Set("ref", get(leaf, "ref")).Set("fc", get(leaf, "fc")).Set("path", path).
			Set("type", TypeName(get(leaf, "mms_type")))
		entry.Set("attributes", append(list(get(entry, "attributes")), attribute))
	}
	leafCount := 0
	sorted := make([]any, 0, len(names))
	sort.Strings(names)
	for _, name := range names {
		entry := objects[name]
		fcs, attributes := list(get(entry, "fcs")), list(get(entry, "attributes"))
		sort.SliceStable(fcs, func(i, j int) bool { return rank(fcs[i]) < rank(fcs[j]) })
		sort.SliceStable(attributes, func(i, j int) bool {
			return rank(get(attributes[i].(*pyjson.Object), "fc")) < rank(get(attributes[j].(*pyjson.Object), "fc"))
		})
		leafCount += len(attributes)
		sorted = append(sorted, entry)
	}
	datasets := []any{}
	members := get(answer, "dataset_members")
	for _, raw := range list(get(answer, "datasets")) {
		name := pytext.Str(raw)
		info, _ := get(asObject(members), name).(*pyjson.Object)
		var found any = []any{}
		if info != nil {
			if value, ok := info.Get("members"); ok {
				found = value
			}
		}
		datasets = append(datasets, pyjson.NewObject().Set("name", raw).Set("ref", reference+"$"+name).Set("members", found))
	}
	reports := []any{}
	for _, pair := range [][2]string{{"BR", "buffered_reports"}, {"RP", "unbuffered_reports"}} {
		for _, raw := range list(get(answer, pair[1])) {
			reports = append(reports, pyjson.NewObject().Set("name", raw).
				Set("ref", reference+"."+pair[0]+"."+pytext.Str(raw)).Set("buffered", pair[0] == "BR"))
		}
	}
	name, found := answer.Get("name")
	if !found {
		name = reference[strings.LastIndexByte(reference, '/')+1:]
	}
	return pyjson.NewObject().Set("ref", reference).Set("name", name).Set("objects", sorted).
		Set("datasets", datasets).Set("reports", reports).Set("leaf_count", int64(leafCount))
}

func asObject(value any) *pyjson.Object {
	object, _ := value.(*pyjson.Object)
	return object
}

func contains(items []any, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
