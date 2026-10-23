package scl

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/pyjson"
)

// readGoldenList loads a golden JSON list of {input: base64, result} objects.
func readGoldenList(t *testing.T, name string) []*pyjson.Object {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "golden", name))
	if err != nil {
		t.Skipf("golden %s missing: %v", name, err)
	}
	decoded, err := pyjson.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	var out []*pyjson.Object
	for _, item := range decoded.([]any) {
		out = append(out, item.(*pyjson.Object))
	}
	return out
}

func text(o *pyjson.Object, key string) string {
	value, _ := o.Get(key)
	s, _ := value.(string)
	return s
}

// TestParseDocumentMatchesPython replays parse_document over the hand-written malformed
// inputs and the 3000 random mutations recorded from CPython (expat 2.6.3, defusedxml).
func TestParseDocumentMatchesPython(t *testing.T) {
	asPythonTwin(t)
	cases := readGoldenList(t, "xml_errors.json")
	failures := 0
	for index, item := range cases {
		input, err := base64.StdEncoding.DecodeString(text(item, "input"))
		if err != nil {
			t.Fatal(err)
		}
		want := text(item, "result")
		got := "OK"
		if _, err := ParseDocument(input, false); err != nil {
			got = err.Error()
		}
		if got != want {
			failures++
			if failures <= 40 {
				t.Errorf("case %d %q:\n got  %s\n want %s", index, clip(input), got, want)
			}
		}
	}
	if failures > 0 {
		t.Errorf("%d of %d cases differ", failures, len(cases))
	}
}

func clip(b []byte) string {
	if len(b) > 120 {
		return string(b[:120]) + "..."
	}
	return string(b)
}
