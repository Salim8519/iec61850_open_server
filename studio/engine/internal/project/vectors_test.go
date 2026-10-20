package project

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// vectorFixtures are the IED models and SCL texts the golden vectors refer to.
type vectorFixtures struct {
	models  map[string]*pyjson.Object
	sources map[string]string
}

func loadJSON(t *testing.T, path string) any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	value, err := pyjson.Decode(data)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return value
}

func loadFixtures(t *testing.T) *vectorFixtures {
	t.Helper()
	root := loadJSON(t, filepath.Join("testdata", "generated", "fixtures.json")).(*pyjson.Object)
	fx := &vectorFixtures{models: map[string]*pyjson.Object{}, sources: map[string]string{}}
	models, _ := root.Get("models")
	for _, name := range models.(*pyjson.Object).Keys() {
		model, _ := models.(*pyjson.Object).Get(name)
		fx.models[name] = model.(*pyjson.Object)
	}
	sources, _ := root.Get("sources")
	for _, name := range sources.(*pyjson.Object).Keys() {
		text, _ := sources.(*pyjson.Object).Get(name)
		fx.sources[name] = text.(string)
	}
	scd, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "scd", "model_substation.scd"))
	if err != nil {
		t.Fatal(err)
	}
	// Python read_text() translates line endings; the source text is what import_scl kept.
	fx.sources["FEED1"] = strings.ReplaceAll(strings.ReplaceAll(string(scd), "\r\n", "\n"), "\r", "\n")
	return fx
}

// expand replaces "@source:NAME" and {"@repeat": v, "count": n} exactly as vectors_gen.py.
func (fx *vectorFixtures) expand(value any) any {
	switch x := value.(type) {
	case string:
		if name, ok := strings.CutPrefix(x, "@source:"); ok {
			return fx.sources[name]
		}
	case []any:
		items := make([]any, len(x))
		for index, item := range x {
			items[index] = fx.expand(item)
		}
		return items
	case *pyjson.Object:
		if x.Len() == 2 && pyval.Has(x, "@repeat") && pyval.Has(x, "count") {
			item, _ := x.Get("@repeat")
			count, _ := x.Get("count")
			items := make([]any, count.(int64))
			for index := range items {
				items[index] = fx.expand(item)
			}
			return items
		}
		result := pyjson.NewObject()
		for _, key := range x.Keys() {
			item, _ := x.Get(key)
			result.Set(key, fx.expand(item))
		}
		return result
	}
	return value
}

// errorObject is the vector form of an error.
func errorObject(err error) *pyjson.Object {
	result := pyjson.NewObject()
	switch e := err.(type) {
	case *ContractError:
		return result.Set("class", "ValidationError").Set("details", e.Details)
	case *apperr.Error:
		class := map[apperr.Kind]string{apperr.Studio: "StudioError", apperr.Value: "ValueError",
			apperr.Key: "KeyError", apperr.Runtime: "RuntimeError"}[e.Kind]
		result.Set("class", class).Set("message", e.Message)
		if e.Kind != apperr.Studio {
			return result.Set("code", nil).Set("details", nil)
		}
		details := e.Details
		if details == nil {
			details = []any{}
		}
		return result.Set("code", e.Code).Set("details", details)
	case *pyval.Error:
		return result.Set("class", e.Class).Set("message", e.Message).Set("code", nil).Set("details", nil)
	}
	return result.Set("class", "Go error").Set("message", err.Error())
}

func TestGoldenVectors(t *testing.T) {
	fx := loadFixtures(t)
	files, _ := filepath.Glob(filepath.Join("testdata", "generated", "vectors_*.json"))
	if len(files) == 0 {
		t.Fatal("no vector files")
	}
	total := 0
	for _, file := range files {
		for _, entry := range loadJSON(t, file).([]any) {
			vector := entry.(*pyjson.Object)
			total++
			name, _ := vector.Get("name")
			t.Run(filepath.Base(file)+"/"+name.(string), func(t *testing.T) { checkVector(t, fx, vector) })
		}
	}
	t.Logf("%d vectors", total)
}

func checkVector(t *testing.T, fx *vectorFixtures, vector *pyjson.Object) {
	fn, _ := vector.Get("fn")
	args, _ := vector.Get("args")
	names, _ := vector.Get("ieds")
	ieds := []*pyjson.Object{}
	for _, name := range names.([]any) {
		ieds = append(ieds, pyval.CopyObject(fx.models[name.(string)]))
	}
	run, found := runners[fn.(string)]
	if !found {
		t.Fatalf("no Go runner for %s", fn)
	}
	result, err := run(fx.expand(args).(*pyjson.Object), ieds, fx)
	if expected, isError := vector.Get("err"); isError {
		if err == nil {
			t.Fatalf("expected error %s, got result", dumps(t, expected))
		}
		if got, want := pythonRecorded(dumps(t, errorObject(err))), dumps(t, expected); got != want {
			t.Fatalf("error mismatch\n got: %s\nwant: %s", got, want)
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error %s", dumps(t, errorObject(err)))
	}
	got := dumps(t, result)
	if want, ok := vector.Get("out"); ok && got != want.(string) {
		t.Fatalf("output mismatch\n got: %s\nwant: %s", got, want)
	}
	if want, ok := vector.Get("out_sha256"); ok {
		digest := sha256.Sum256([]byte(got))
		if hex.EncodeToString(digest[:]) != want.(string) {
			head, _ := vector.Get("out_head")
			t.Fatalf("output digest mismatch\n got: %.300s\nwant: %s", got, head)
		}
	}
}

func dumps(t *testing.T, value any) string {
	t.Helper()
	data, err := pyjson.Dumps(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes.TrimSpace(data))
}
