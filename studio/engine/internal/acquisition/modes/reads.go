package modes

import (
	"sort"
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// GocbObject is gocb_object: "LD/LLN0$GO$name" (SCL/MMS style) becomes "LD/LLN0.name"
// (client object reference); only the first "$GO$" is replaced.
func GocbObject(gocbRef string) string {
	return strings.Replace(gocbRef, "$GO$", ".", 1)
}

// GocbItems is gocb_items: the GO-constrained read items for every attribute of every
// GOOSE control block the IED publishes.
func GocbItems(ied *pyjson.Object) []any {
	items := []any{}
	for _, control := range pyv.Objects(pyv.GetList(pyv.OrObj(ied, "goose"), "publishes")) {
		base := GocbObject(pyv.GetStr(control, "gocb_ref"))
		for _, name := range GocbAttributes() {
			items = append(items, pyv.O("ref", base+"."+name, "fc", "GO"))
		}
	}
	return items
}

// ModeItems is mode_items: Mod/Beh of every logical device and LPHD.Sim, only where the
// SCD model has them as ST points.
func ModeItems(ied *pyjson.Object) []any {
	refs := map[string]bool{}
	domainSet := map[string]bool{}
	for _, signal := range pyv.Objects(pyv.GetList(ied, "signals")) {
		if !pyv.Equal(pyv.Get(signal, "fc"), "ST") {
			continue
		}
		ref := pyv.GetStr(signal, "ref")
		refs[ref] = true
		domain, _, _ := strings.Cut(ref, "/")
		domainSet[domain] = true
	}
	var wanted []string
	for _, domain := range sortedKeys(domainSet) {
		wanted = append(wanted, domain+"/LLN0.Mod.stVal", domain+"/LLN0.Beh.stVal")
	}
	for _, ref := range sortedKeys(refs) {
		if isSimulationRef(ref) {
			wanted = append(wanted, ref)
		}
	}
	items := []any{}
	for _, ref := range wanted {
		if refs[ref] {
			items = append(items, pyv.O("ref", ref, "fc", "ST"))
		}
	}
	return items
}

// ParseAnswer is parse_answer: {ref: value} for readable items (a "structure" entry adds
// all its members) and {ref: error} for the rest.
func ParseAnswer(answer *pyjson.Object) (*pyjson.Object, *pyjson.Object) {
	values, errors := pyjson.NewObject(), pyjson.NewObject()
	for _, entry := range pyv.Objects(pyv.List(pyv.GetOr(answer, "values", []any{}))) {
		switch {
		case pyv.Has(entry, "value"):
			values.Set(pyv.GetStr(entry, "ref"), pyv.Get(entry, "value"))
		case pyv.Has(entry, "structure"):
			if structure := pyv.GetObj(entry, "structure"); structure != nil {
				pyv.Update(values, structure)
			}
		default:
			errors.Set(pyv.GetStr(entry, "ref"), pyv.Get(entry, "error"))
		}
	}
	return values, errors
}

// sortedKeys returns the keys of a set in Python's sorted() order (code point order, which
// is byte order for valid UTF-8).
func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
