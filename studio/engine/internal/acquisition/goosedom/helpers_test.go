package goosedom

import (
	"strings"
	"testing"

	"studio-engine/internal/acquisition/goldentest"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// same compares json.dumps bytes (insertion order kept) with the Python recording.
func same(t *testing.T, c goldentest.Case, got any) {
	t.Helper()
	body, err := pyjson.Dumps(got)
	if err != nil {
		t.Fatalf("%s: encode: %v", c.Name, err)
	}
	goldentest.Text(t, c, string(body))
}

// run calls check for every case of file, by the function name before the first "/".
func run(t *testing.T, file string, checks map[string]func(goldentest.Case) any) {
	t.Helper()
	counts := map[string]int{}
	for _, c := range goldentest.Load(t, file) {
		function, _, _ := strings.Cut(c.Name, "/")
		check, found := checks[function]
		if !found {
			t.Fatalf("%s: no Go check for %s", file, function)
		}
		same(t, c, check(c))
		counts[function]++
	}
	for function := range checks {
		if counts[function] == 0 {
			t.Errorf("%s: no golden cases for %s", file, function)
		}
	}
}

func objs(value any) []*pyjson.Object { return pyv.Objects(pyv.List(value)) }

func updates(items []Update) []any {
	result := []any{}
	for _, item := range items {
		result = append(result, []any{item.Ref, item.Value})
	}
	return result
}

func texts(items []string) []any { return pyv.Strings(items) }
