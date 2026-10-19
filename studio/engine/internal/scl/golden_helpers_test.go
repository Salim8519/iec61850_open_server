package scl

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

const repoRoot = "../../../.."

// seedValidator answers from the XSD verdicts Python recorded (golden/xsd_seed.json).
type seedValidator map[string]SchemaVerdict

func (s seedValidator) Validate(source []byte, schema string) (SchemaVerdict, error) {
	sum := sha256.Sum256(source)
	verdict, ok := s[hex.EncodeToString(sum[:])+"/"+schema]
	if !ok {
		return SchemaVerdict{}, apperr.RuntimeError("no recorded XSD verdict")
	}
	return verdict, nil
}

func loadSeed(t *testing.T) seedValidator {
	t.Helper()
	seed := seedValidator{}
	for _, item := range readGoldenList(t, "xsd_seed.json") {
		verdict := SchemaVerdict{Status: text(item, "status")}
		errorsValue, _ := item.Get("errors")
		for _, raw := range errorsValue.([]any) {
			entry := raw.(*pyjson.Object)
			line, _ := entry.Get("line")
			verdict.Errors = append(verdict.Errors, SchemaError{Line: int(line.(int64)), Message: text(entry, "message")})
		}
		seed[text(item, "sha256")+"/"+text(item, "schema")] = verdict
	}
	return seed
}

// errorJSON renders an error like golden_common.py: {type, message, code, details}.
func errorJSON(err error) string {
	o := pyjson.NewObject()
	var app *apperr.Error
	switch {
	case errors.As(err, &app) && app.Kind == apperr.Studio:
		details := app.Details
		if details == nil {
			details = []any{}
		}
		o.Set("type", "StudioError").Set("message", app.Message).Set("code", app.Code).Set("details", details)
	case errors.As(err, &app) && app.Kind == apperr.Value:
		kind := "ValueError"
		if strings.HasPrefix(app.Message, "'utf-8' codec can't decode") {
			kind = "UnicodeDecodeError"
		}
		o.Set("type", kind).Set("message", app.Message).Set("code", nil).Set("details", nil)
	case errors.Is(err, errTypeError):
		o.Set("type", "TypeError").Set("message", err.Error()).Set("code", nil).Set("details", nil)
	default:
		o.Set("type", "GoError").Set("message", err.Error()).Set("code", nil).Set("details", nil)
	}
	data, _ := pyjson.Dumps(o)
	return string(data)
}

// normalise replaces ids and sources as golden_gen.py does and returns json.dumps text.
func normalise(models []*pyjson.Object, cfgs map[string]string, prefix string) string {
	list := []any{}
	for index, model := range models {
		copyModel := pyjson.NewObject()
		for _, key := range model.Keys() {
			value, _ := model.Get(key)
			switch key {
			case "id":
				value = "ied-" + strconv.Itoa(index)
			case "source":
				sum := sha256.Sum256([]byte(value.(string)))
				value = "sha256:" + hex.EncodeToString(sum[:])
			case "cfg":
				if cfgs != nil {
					cfgs[prefix+strconv.Itoa(index)+".cfg"] = value.(string)
				}
			}
			copyModel.Set(key, value)
		}
		list = append(list, copyModel)
	}
	data, _ := pyjson.Dumps(list)
	return string(data)
}

func readFile(t *testing.T, path string) (string, bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// firstDiff returns a short window around the first differing byte.
func firstDiff(got, want string) string {
	i := 0
	for i < len(got) && i < len(want) && got[i] == want[i] {
		i++
	}
	lo := i - 80
	if lo < 0 {
		lo = 0
	}
	clipAt := func(s string) string {
		hi := i + 120
		if hi > len(s) {
			hi = len(s)
		}
		if lo > len(s) {
			return ""
		}
		return s[lo:hi]
	}
	return "at byte " + strconv.Itoa(i) + "\n  got:  " + clipAt(got) + "\n  want: " + clipAt(want)
}

func goldenDir(name string) string { return filepath.Join("testdata", "golden", name) }
