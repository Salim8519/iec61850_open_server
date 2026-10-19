package flow

import (
	"sort"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// BuildGraph is build_graph(flow, loaded, ied): {ieds, blocks, edges, diagnostics,
// summary}, only around one IED when ied is not "" (Python's None).
func BuildGraph(flow *scl.Flow, loaded map[string]bool, ied string) *pyjson.Object {
	names := map[string]bool{}
	for _, item := range flow.IEDs {
		names[str(item, "name")] = true
	}
	g := &graph{}
	g.goose(flow)
	g.reports(flow, names)
	edges := dedupe(g.edges)
	sort.SliceStable(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		for _, key := range []string{"type", "publisher", "id"} {
			if str(a, key) != str(b, key) {
				return str(a, key) < str(b, key)
			}
		}
		return false
	})
	blocks, diagnostics := g.blocks, g.diagnostics
	if ied != "" {
		var kept []*pyjson.Object
		used := map[string]bool{}
		for _, edge := range edges {
			if str(edge, "publisher") == ied || str(edge, "subscriber") == ied {
				kept = append(kept, edge)
				used[str(edge, "block")] = true
			}
		}
		edges = kept
		blocks = filter(blocks, func(block *pyjson.Object) bool { return used[str(block, "id")] || str(block, "ied") == ied })
		diagnostics = filter(diagnostics, func(item *pyjson.Object) bool {
			return str(item, "ied") == ied || namesWord(str(item, "message"), ied)
		})
	}
	sort.SliceStable(blocks, func(i, j int) bool {
		a, b := blocks[i], blocks[j]
		for _, key := range []string{"type", "ied", "ref"} {
			if str(a, key) != str(b, key) {
				return str(a, key) < str(b, key)
			}
		}
		return false
	})
	ieds := make([]*pyjson.Object, 0, len(flow.IEDs))
	for _, item := range flow.IEDs {
		copied := pyjson.NewObject()
		for _, key := range item.Keys() {
			copied.Set(key, get(item, key))
		}
		ieds = append(ieds, copied.Set("loaded", loaded[str(item, "name")]))
	}
	sort.SliceStable(ieds, func(i, j int) bool { return str(ieds[i], "name") < str(ieds[j], "name") })
	if ied != "" {
		involved := map[string]bool{ied: true}
		for _, edge := range edges {
			involved[str(edge, "publisher")], involved[str(edge, "subscriber")] = true, true
		}
		ieds = filter(ieds, func(item *pyjson.Object) bool { return involved[str(item, "name")] })
	}
	return pyjson.NewObject().Set("ieds", objects(ieds)).Set("blocks", objects(blocks)).Set("edges", objects(edges)).
		Set("diagnostics", objects(diagnostics)).Set("summary", summary(ieds, edges, diagnostics))
}

func summary(ieds, edges, diagnostics []*pyjson.Object) *pyjson.Object {
	counts := map[string]int64{}
	for _, edge := range edges {
		counts["edge "+str(edge, "type")]++
	}
	for _, item := range diagnostics {
		counts["level "+str(item, "level")]++
	}
	return pyjson.NewObject().Set("ieds", int64(len(ieds))).Set("goose_links", counts["edge goose"]).
		Set("report_links", counts["edge report"]).Set("problems", counts["level problem"]).
		Set("infos", counts["level info"])
}

// dedupe is list({edge["id"]: edge for edge in edges}.values()).
func dedupe(edges []*pyjson.Object) []*pyjson.Object {
	position := map[string]int{}
	var result []*pyjson.Object
	for _, edge := range edges {
		if index, seen := position[str(edge, "id")]; seen {
			result[index] = edge
			continue
		}
		position[str(edge, "id")] = len(result)
		result = append(result, edge)
	}
	return result
}

func filter(items []*pyjson.Object, keep func(*pyjson.Object) bool) []*pyjson.Object {
	var result []*pyjson.Object
	for _, item := range items {
		if keep(item) {
			result = append(result, item)
		}
	}
	return result
}

func objects(items []*pyjson.Object) []any {
	result := make([]any, len(items))
	for index, item := range items {
		result[index] = item
	}
	return result
}

// namesWord is re.search(rf"(?<![A-Za-z0-9_]){re.escape(ied)}(?![A-Za-z0-9_])", message).
func namesWord(message, ied string) bool {
	for start := 0; ; {
		index := strings.Index(message[start:], ied)
		if index < 0 {
			return false
		}
		at := start + index
		before := at == 0 || !wordByte(message[at-1])
		after := at+len(ied) == len(message) || !wordByte(message[at+len(ied)])
		if before && after {
			return true
		}
		start = at + 1
		if start > len(message) {
			return false
		}
	}
}

func wordByte(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_'
}
