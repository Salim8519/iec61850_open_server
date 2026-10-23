package scl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// TestGooseSourcesMatchPython replays golden/goose_sources.json (goose_sources_gen.py):
// read_goose_sources over single files, pairs and synthetic multi-file stations, and
// _with_members for every declared subscription, compared byte for byte.
func TestGooseSourcesMatchPython(t *testing.T) {
	asPythonTwin(t)
	data, err := os.ReadFile(filepath.Join("testdata", "golden", "goose_sources.json"))
	if err != nil {
		t.Skip("goose_sources golden missing")
	}
	decoded, err := pyjson.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	texts := map[string]string{}
	compared, failures := 0, 0
	check := func(name, got, want string) {
		compared++
		if got != want {
			failures++
			if failures <= 10 {
				t.Errorf("%s: %s", name, firstDiff(got, want))
			}
		}
	}
	for _, raw := range decoded.([]any) {
		entry := raw.(*pyjson.Object)
		name := text(entry, "name")
		var documents []GooseDocument
		for _, item := range get(entry, "documents").([]any) {
			ref := item.(*pyjson.Object)
			body, inline := ref.Get("text")
			if !inline {
				path := text(ref, "case")
				if _, ok := texts[path]; !ok {
					payload, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(path)))
					if err != nil {
						t.Fatal(err)
					}
					texts[path] = strings.TrimPrefix(string(payload), "\xef\xbb\xbf")
				}
				body = texts[path]
			}
			var loaded []string
			for _, value := range get(ref, "loaded").([]any) {
				loaded = append(loaded, value.(string))
			}
			documents = append(documents, GooseDocument{Text: body.(string), Loaded: loaded})
		}
		result, err := ReadGooseSources(documents)
		check(name, outcomeJSON(result, err), expected(entry))
		for _, item := range get(entry, "members").([]any) {
			member := item.(*pyjson.Object)
			doc, _ := member.Get("doc")
			ied := pyjson.NewObject().Set("source", documents[doc.(int64)].Text).Set("name", text(member, "ied"))
			found, err := SubscriptionMembers(ied, pyjson.NewObject().Set("gocb_ref", text(member, "gocb_ref")))
			check(name+" members "+text(member, "gocb_ref"), outcomeJSON(found, err), expected(member))
		}
	}
	t.Logf("%d Python answers compared, %d differ", compared, failures)
}

func outcomeJSON(value *pyjson.Object, err error) string {
	if err != nil {
		return errorJSON(err)
	}
	data, _ := pyjson.Dumps(value)
	return string(data)
}

func expected(entry *pyjson.Object) string {
	if value, ok := entry.Get("result"); ok {
		data, _ := pyjson.Dumps(value)
		return string(data)
	}
	value, _ := entry.Get("error")
	data, _ := pyjson.Dumps(value)
	return string(data)
}
