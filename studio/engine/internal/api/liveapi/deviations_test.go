package liveapi

import (
	"strings"

	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// goExpected maps what the frozen Python engine recorded to what Go answers, for the
// documented Go-only deviations only; every other byte must still match.
//
// ADR-0114: /goose/links counts an unheard link from a publisher outside the project as
// "remote" instead of "not_heard", so the summary is recomputed with the Go rule from the
// recorded links and IEDs.
func goExpected(path, recorded string) string {
	if !strings.HasPrefix(path, "/api/v1/goose/links") {
		return recorded
	}
	decoded, err := pyjson.Decode([]byte(recorded))
	object, ok := decoded.(*pyjson.Object)
	if err != nil || !ok || object.Len() == 0 {
		return recorded
	}
	if _, has := object.Get("summary"); !has {
		return recorded
	}
	outside := map[string]bool{}
	for _, ied := range pyv.Objects(pyv.GetList(object, "ieds")) {
		if pyv.Equal(pyv.Get(ied, "in_project"), false) {
			outside[pyv.PyStr(pyv.Get(ied, "name"))] = true
		}
	}
	object.Set("summary", goosedom.Summary(pyv.Objects(pyv.GetList(object, "links")), outside))
	encoded, err := pyjson.Flask(object)
	if err != nil {
		return recorded
	}
	return string(encoded)
}
