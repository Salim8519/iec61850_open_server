package scl

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"studio-engine/internal/pyjson"
)

// TestCorpusMatchesPython replays every public fixture and corpus case through preview,
// validate and import and compares bytes with the goldens CPython produced.
func TestCorpusMatchesPython(t *testing.T) {
	asPythonTwin(t)
	manifest := readGoldenList(t, "manifest.json")
	importer := &Importer{Validator: loadSeed(t)}
	compared := 0
	defer func() { t.Logf("%d golden files compared over %d cases", compared, len(manifest)) }()
	failures := 0
	report := func(name, file, detail string) {
		failures++
		if failures <= 30 {
			t.Errorf("%s/%s: %s", name, file, detail)
		}
	}
	for _, entry := range manifest {
		name := text(entry, "case")
		input, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(text(entry, "input"))))
		if err != nil {
			t.Fatal(err)
		}
		dir := goldenDir(name)
		compare := func(file, got string) {
			compared++
			want, ok := readFile(t, filepath.Join(dir, file))
			if !ok {
				report(name, file, "golden missing; Go gave "+clip([]byte(got)))
				return
			}
			if got != want {
				report(name, file, firstDiff(got, want))
			}
		}
		if models, err := Preview(input); err != nil {
			compare("error.json", errorJSON(err))
		} else {
			data, _ := pyjson.Dumps(objects(models))
			compare("preview.json", string(data))
		}
		if result, err := Validate(input, importer.Validator); err != nil {
			compare("validate.json", errorJSON(err))
		} else {
			data, _ := pyjson.Dumps(result)
			compare("validate.json", string(data))
		}
		cfgs := map[string]string{}
		if models, err := importer.Import(input, "case.scd", SelectAll); err != nil {
			compare("import-all.json", errorJSON(err))
		} else {
			compare("import-all.json", normalise(models, cfgs, "cfg-"))
		}
		names, _ := entry.Get("ied_names")
		aliases, _ := entry.Get("aliases")
		for n, raw := range names.([]any) {
			if n >= 3 {
				break
			}
			file := "import-each-" + strconv.Itoa(n) + ".json"
			if target, ok := aliases.(*pyjson.Object).Get(file); ok {
				file = target.(string)
			}
			if models, err := importer.Import(input, "case.scd", Selection{Names: []any{raw}}); err != nil {
				compare(file, errorJSON(err))
			} else {
				compare(file, normalise(models, cfgs, "cfg-each-"+strconv.Itoa(n)+"-"))
			}
		}
		for file, cfg := range cfgs {
			if want, ok := readFile(t, filepath.Join(dir, file)); ok && want != cfg {
				report(name, file, firstDiff(cfg, want))
			}
		}
	}
	if failures > 0 {
		t.Errorf("%d differences over %d cases", failures, len(manifest))
	}
}
