package masterpoints

import (
	"strconv"
	"strings"

	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/pyjson"
)

// FromGateway plans list points from the project's gateway point list of the same protocol:
// only the address, the type as the master sees it (addpoints.MasterOfGatewayPoint) and the name.
// No IED is linked: "from" keeps the gateway point's protocol, address and kind, which the
// optional Check against IED uses to find that gateway point again. Points already copied are
// left out, and a point whose address the list already uses is left out with a note. iedNames
// (IED id to name) words the names; a stored gateway point keeps only the IED id.
func FromGateway(protocol string, gatewayPoints []any, current []any, iedNames map[string]string) ([]Result, []string) {
	copied := map[string]bool{}
	for _, raw := range current {
		if from, ok := field(raw.(*pyjson.Object), "from").(*pyjson.Object); ok {
			copied[fromKey(text(from, "protocol"), field(from, "address"), text(from, "kind"))] = true
		}
	}
	plan, results, notes, already := newPlanner(protocol, current), []Result{}, []string{}, 0
	for _, raw := range gatewayPoints {
		source, ok := raw.(*pyjson.Object)
		kind := text(source, "kind")
		address, isInt := field(source, "address").(int64)
		entry, known := addpoints.MasterOfGatewayPoint(protocol, kind, text(source, "format"))
		if !ok || !isInt || !known {
			continue
		}
		if copied[fromKey(protocol, address, kind)] {
			already++
			continue
		}
		from := pyjson.NewObject().Set("protocol", protocol).Set("address", address).Set("kind", kind)
		name := gatewayName(source, iedNames)
		point, problem := plan.add(pyjson.NewObject().Set("name", name).Set("type", entry.Key).
			Set("address", address).Set("from", from))
		if problem != "" {
			notes = append(notes, name+": "+problem+", so it was left out.")
			continue
		}
		results = append(results, Result{Number: len(results) + 1, Point: point})
	}
	if already > 0 {
		notes = append([]string{strconv.Itoa(already) + " gateway points are already in your list."}, notes...)
	}
	return results, notes
}

func fromKey(protocol string, address any, kind string) string {
	number, _ := address.(int64)
	return protocol + ":" + strconv.FormatInt(number, 10) + ":" + kind
}

// gatewayName is the name a copied point gets, as the gateway list shows it: the IED name and the
// point's name ("FEED1 CSWI1 Pos"; the short reference when it has no name), at most 120
// characters. The IED name is words only: the copied point is not linked to that IED.
func gatewayName(point *pyjson.Object, iedNames map[string]string) string {
	name, ied := strings.TrimSpace(text(point, "description")), text(point, "ied_name")
	if known := iedNames[text(point, "ied")]; known != "" {
		ied = known
	}
	if name == "" {
		if ref := text(point, "ref"); ref != "" {
			name = addpoints.ShortName(ref)
		}
	}
	if ied != "" && !strings.HasPrefix(strings.ToLower(name), strings.ToLower(ied)) {
		name = strings.TrimSpace(ied + " " + name)
	}
	runes := []rune(name)
	if len(runes) > 120 {
		name = string(runes[:120])
	}
	return name
}

// FromOf is the gateway point a list point was copied from: (protocol, address, kind).
func FromOf(point *pyjson.Object) (string, int64, string, bool) {
	from, ok := field(point, "from").(*pyjson.Object)
	if !ok {
		return "", 0, "", false
	}
	address, _ := field(from, "address").(int64)
	return text(from, "protocol"), address, text(from, "kind"), true
}
