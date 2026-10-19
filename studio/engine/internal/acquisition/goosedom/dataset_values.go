package goosedom

import (
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// DatasetValues maps received GOOSE dataset values to the publisher's SCD signal
// references (dataset_values). A structured member is spread over the signals below
// its FCDA reference with the member's functional constraint, only when the counts
// match. The result is a new object in Python insertion order.
func DatasetValues(ied, control *pyjson.Object, values any) *pyjson.Object {
	result := pyjson.NewObject()
	var dataset *pyjson.Object
	for _, item := range objects(ied, "datasets") {
		if pyv.Equal(pyv.Get(item, "ref"), pyv.Get(control, "dataset_ref")) {
			dataset = item
			break
		}
	}
	items, isList := values.([]any)
	if dataset == nil || !isList || len(items) != len(pyv.List(pyv.GetOr(dataset, "fcda_refs", []any{}))) {
		return result
	}
	members := pyv.GetList(control, "members")
	fcdaRefs := pyv.GetList(dataset, "fcda_refs")
	count := min(len(members), len(fcdaRefs), len(items))
	var byFC map[string][]string
	for position := 0; position < count; position++ {
		fcdaRef := pyv.PyStr(fcdaRefs[position])
		value := items[position]
		fc := memberFC(pyv.Str(members[position]))
		nested, structured := value.([]any)
		if !structured {
			result.Set(fcdaRef, value)
			continue
		}
		if byFC == nil {
			byFC = signalsByFC(objects(ied, "signals"))
		}
		var targets []string
		for _, ref := range byFC[fc] {
			if strings.HasPrefix(ref, fcdaRef+".") || strings.HasPrefix(ref, fcdaRef+"(") {
				targets = append(targets, ref)
			}
		}
		leaves := flatten(nested, nil)
		if len(targets) == len(leaves) {
			for index, target := range targets {
				result.Set(target, leaves[index])
			}
		}
	}
	return result
}

// memberFC is the functional constraint of a dataset member "LN$FC$DO$DA" ("ST" when
// the member has no "$").
func memberFC(member string) string {
	if slash := strings.Index(member, "/"); slash >= 0 {
		member = member[slash+1:]
	}
	parts := strings.Split(member, "$")
	if len(parts) > 1 {
		return parts[1]
	}
	return "ST"
}

// signalsByFC indexes signal references by functional constraint once, in signal order.
func signalsByFC(signals []*pyjson.Object) map[string][]string {
	index := map[string][]string{}
	for _, signal := range signals {
		fc, isText := pyv.Get(signal, "fc").(string)
		ref, isRef := pyv.Get(signal, "ref").(string)
		if isText && isRef {
			index[fc] = append(index[fc], ref)
		}
	}
	return index
}

// flatten is list(_leaves(value)): nested lists read depth first.
func flatten(items []any, leaves []any) []any {
	for _, item := range items {
		if nested, ok := item.([]any); ok {
			leaves = flatten(nested, leaves)
		} else {
			leaves = append(leaves, item)
		}
	}
	return leaves
}
