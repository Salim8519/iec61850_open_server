package gateway

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pycsv"
	"studio-engine/internal/tools/pyoracle"
)

// oracleSeed is the random seed every oracle script starts from; GATEWAY_ORACLE_SEED
// replays other random inputs (for a deeper search before a change is recorded).
func oracleSeed() string {
	if seed := os.Getenv("GATEWAY_ORACLE_SEED"); seed != "" {
		return seed
	}
	return "61850"
}

// replay runs one oracle script and hands every case except fixtures to run; it fails when
// the script printed no case and reports how many cases each function had.
func replay(t *testing.T, script, count string, run func(item pyoracle.Case, fixtures map[string][]*pyjson.Object)) {
	t.Helper()
	cases := pyoracle.Cases(t, filepath.Join("testdata", script), nil, oracleSeed(), count)
	if len(cases) == 0 {
		t.Fatal("no oracle cases")
	}
	fixtures, counts := map[string][]*pyjson.Object{}, map[string]int{}
	for _, item := range cases {
		if item.Fn == "fixture" {
			fixtures[item.Args[0].(string)] = objects(item.Result)
			continue
		}
		counts[item.Fn]++
		run(item, fixtures)
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Logf("%s: %d cases", name, counts[name])
	}
}

// errText is the oracle's "<Class>: <text>" for an error the port returned.
func errText(err error) string {
	var value *apperr.Error
	if errors.As(err, &value) {
		return "ValueError: " + value.Message
	}
	var csvError *pycsv.Error
	if errors.As(err, &csvError) {
		return "Error: " + csvError.Text
	}
	return err.Error()
}

// expect compares one result (or error) with the Python case.
func expect(t *testing.T, item pyoracle.Case, got any, err error) {
	t.Helper()
	if item.Error != "" {
		if err == nil || errText(err) != item.Error {
			t.Errorf("%s %s: got %v / %s, want error %s", item.Fn, short(t, item.Args), err, short(t, got), item.Error)
		}
		return
	}
	if err != nil {
		t.Errorf("%s %s: unexpected error %s", item.Fn, short(t, item.Args), errText(err))
		return
	}
	if gotText, want := pyoracle.Dumps(t, got), pyoracle.Dumps(t, item.Result); gotText != want {
		at := 0
		for at < len(gotText) && at < len(want) && gotText[at] == want[at] {
			at++
		}
		from := max(0, at-200)
		t.Errorf("%s %s:\n got ...%s\nwant ...%s", item.Fn, short(t, item.Args), clip(gotText[from:]), clip(want[from:]))
	}
}

func short(t *testing.T, value any) string { return clip(pyoracle.Dumps(t, value)) }

func clip(text string) string {
	if len(text) > 1500 {
		return text[:1500] + "..."
	}
	return text
}

func objects(value any) []*pyjson.Object {
	list, _ := value.([]any)
	result := make([]*pyjson.Object, len(list))
	for index, item := range list {
		result[index], _ = item.(*pyjson.Object)
	}
	return result
}

func object(value any) *pyjson.Object {
	result, _ := value.(*pyjson.Object)
	return result
}

func list(value any) []any {
	result, _ := value.([]any)
	return result
}

func strs(values []string) []any {
	if values == nil {
		return nil
	}
	return anyStrings(values)
}

func stringList(value any) []string {
	result := []string{}
	for _, item := range list(value) {
		result = append(result, item.(string))
	}
	return result
}
