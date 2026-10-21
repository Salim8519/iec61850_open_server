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
		"with_live": func(c goldentest.Case) any { return WithLive(c.Obj(0), objs(c.Args[1]), c.Args[2] == true) },
	})
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
