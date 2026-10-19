package explorer

import (
	"regexp"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Key-point watch limits (domain/explorer_watch.py).
const (
	MaxNodes  = 16
	MaxPoints = 40
)

var keyClasses = map[string]bool{"XCBR": true, "XSWI": true, "CSWI": true, "MMXU": true, "MMXN": true,
	"MSQI": true, "PTRC": true, "LLN0": true, "LPHD": true}

var perNode = map[string]int{"MMXU": 8, "MMXN": 4, "MSQI": 4}

var wanted = map[string]map[string][]string{
	"XCBR": {"Pos": {"stVal", "q"}}, "XSWI": {"Pos": {"stVal", "q"}}, "CSWI": {"Pos": {"stVal"}},
	"PTRC": {"Tr": {"general"}, "Op": {"general"}, "Str": {"general"}},
	"LLN0": {"Beh": {"stVal"}, "Health": {"stVal"}}, "LPHD": {"PhyHealth": {"stVal"}},
}

// classPattern is r"([A-Z]{4})\d*$": Python's \d is any Unicode digit and $ also matches
// before a final newline. Both engines return the leftmost match.
var classPattern = regexp.MustCompile(`([A-Z]{4})\p{Nd}*\n?\z`)

// NodeClass is node_class: the LN class (prefix + 4 letters + instance), or LLN0.
func NodeClass(name string) string {
	if name == "LLN0" {
		return "LLN0"
	}
	if match := classPattern.FindStringSubmatch(name); match != nil {
		return match[1]
	}
	return ""
}

// KeyNodes is key_nodes: "device/node" for key classes, at most 16.
func KeyNodes(device string, names []string) []string {
	var chosen []string
	for _, name := range names {
		if keyClasses[NodeClass(name)] {
			chosen = append(chosen, device+"/"+name)
		}
	}
	if len(chosen) > MaxNodes {
		chosen = chosen[:MaxNodes]
	}
	return chosen
}

// KeyPoints is key_points: watch items (ref, fc, type) from one NodeView.
func KeyPoints(view *pyjson.Object) []any {
	kind := NodeClass(pytext.Str(get(view, "name")))
	var points []any
	if limit, measured := perNode[kind]; measured {
		for _, item := range list(get(view, "objects")) {
			for _, raw := range list(get(item.(*pyjson.Object), "attributes")) {
				attribute := raw.(*pyjson.Object)
				if get(attribute, "fc") == "MX" && strings.HasSuffix(pytext.Str(get(attribute, "path")), "mag.f") {
					points = append(points, item_(attribute))
				}
			}
		}
		if len(points) > limit {
			points = points[:limit]
		}
		return points
	}
	paths := wanted[kind]
	for _, item := range list(get(view, "objects")) {
		accepted := paths[pytext.Str(get(item.(*pyjson.Object), "name"))]
		for _, raw := range list(get(item.(*pyjson.Object), "attributes")) {
			attribute := raw.(*pyjson.Object)
			if get(attribute, "fc") == "ST" && containsText(accepted, get(attribute, "path")) {
				points = append(points, item_(attribute))
			}
		}
	}
	return points
}

func containsText(items []string, value any) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func item_(attribute *pyjson.Object) *pyjson.Object {
	kind, found := attribute.Get("type")
	if !found {
		kind = ""
	}
	return pyjson.NewObject().Set("ref", get(attribute, "ref")).Set("fc", get(attribute, "fc")).Set("type", kind)
}

// MergeWatch is merge: the first item per (ref, fc), at most 40.
func MergeWatch(points []any) []any {
	seen := map[[2]string]bool{}
	result := []any{}
	for _, raw := range points {
		point := raw.(*pyjson.Object)
		key := [2]string{pytext.Repr(get(point, "ref")), pytext.Repr(get(point, "fc"))}
		if !seen[key] {
			seen[key] = true
			result = append(result, point)
		}
	}
	if len(result) > MaxPoints {
		result = result[:MaxPoints]
	}
	return result
}
