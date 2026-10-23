package goosedom

import (
	"testing"

	"studio-engine/internal/acquisition/goldentest"
	"studio-engine/internal/pyjson"
)

func TestLinkMapMatchesPython(t *testing.T) {
	run(t, "links_map.json", map[string]func(goldentest.Case) any{
		"build_link_map": func(c goldentest.Case) any {
			return BuildLinkMap(DeclarationsFromJSON(c.Obj(0)), objs(c.Args[1]))
		},
	})
}

func TestWithLiveMatchesPython(t *testing.T) {
	run(t, "links_live.json", map[string]func(goldentest.Case) any{
		"with_live": func(c goldentest.Case) any {
			// ADR-0114: Python counts an unheard link from a publisher outside the project as
			// "not_heard"; Go counts it as "remote". Compare the summary Python would give.
			result := WithLive(c.Obj(0), objs(c.Args[1]), c.Args[2] == true)
			return result.Set("summary", Summary(objectsOf(result), nil))
		},
	})
}

func objectsOf(result *pyjson.Object) []*pyjson.Object {
	links, _ := result.Get("links")
	var out []*pyjson.Object
	for _, item := range links.([]any) {
		out = append(out, item.(*pyjson.Object))
	}
	return out
}

func TestWithLiveKeepsInput(t *testing.T) {
	link := pyjson.NewObject().Set("status", "declared").Set("gocb_ref", "g").Set("publisher", "P").
		Set("subscriber", "S")
	linkMap := pyjson.NewObject().Set("links", []any{link})
	result := WithLive(linkMap, nil, false)
	if _, found := link.Get("live"); found {
		t.Fatal("WithLive modified the input link")
	}
	if _, found := linkMap.Get("summary"); found || result == linkMap {
		t.Fatal("WithLive modified the input map")
	}
}
