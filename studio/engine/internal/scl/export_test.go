package scl

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
)

// TestExportMatchesPython replays export_ied for every recorded override set: payload
// bytes (lxml serialisation) or the exact error.
func TestExportMatchesPython(t *testing.T) {
	asPythonTwin(t)
	importer := &Importer{Validator: loadSeed(t)}
	compared, failures := 0, 0
	check := func(label, got, want string) {
		compared++
		if got != want {
			failures++
			if failures <= 20 {
				t.Errorf("%s: %s", label, firstDiff(got, want))
			}
		}
	}
	run := func(source, name string, overrides any) string {
		payload, err := importer.Export(source, name, overrides)
		if err != nil {
			return errorJSON(err)
		}
		return string(payload)
	}
	for _, entry := range readGoldenList(t, "manifest.json") {
		caseName := text(entry, "case")
		input, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(text(entry, "input"))))
		if err != nil {
			t.Fatal(err)
		}
		source, err := pyrules.DecodeUTF8Sig(input)
		if err != nil {
			continue
		}
		names, _ := entry.Get("ied_names")
		files, _ := entry.Get("files")
		dir := goldenDir(caseName)
		for _, raw := range files.([]any) {
			file := raw.(string)
			if !strings.HasPrefix(file, "export-") || !strings.HasSuffix(file, ".overrides.json") {
				continue
			}
			n, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(file, "export-"), ".overrides.json"))
			data, _ := os.ReadFile(filepath.Join(dir, file))
			overrides, _ := pyjson.Decode(data)
			got := run(source, names.([]any)[n].(string), overrides)
			want, ok := readFile(t, filepath.Join(dir, "export-"+strconv.Itoa(n)+".scd"))
			if !ok {
				want, _ = readFile(t, filepath.Join(dir, "export-"+strconv.Itoa(n)+".error.json"))
			}
			check(caseName+"/"+file, got, want)
		}
		errorsData, err := os.ReadFile(filepath.Join(dir, "export-errors.json"))
		if err != nil {
			continue
		}
		list, _ := pyjson.Decode(errorsData)
		for k, item := range list.([]any) {
			probe := item.(*pyjson.Object)
			overrides, _ := probe.Get("overrides")
			got := run(source, text(probe, "ied"), overrides)
			want := ""
			if failure, ok := probe.Get("error"); ok {
				data, _ := pyjson.Dumps(failure)
				want = string(data)
			} else {
				want, _ = readFile(t, filepath.Join(dir, "export-errors-"+strconv.Itoa(k)+".scd"))
			}
			check(caseName+"/export-errors["+strconv.Itoa(k)+"]", got, want)
		}
	}
	t.Logf("%d exports compared", compared)
	if failures > 0 {
		t.Errorf("%d of %d exports differ", failures, compared)
	}
}
