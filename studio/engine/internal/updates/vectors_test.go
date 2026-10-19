package updates

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"studio-engine/internal/licensing"
	"studio-engine/internal/pyjson"
)

type updateVectors struct {
	TestKey struct{ Kid, Public string } `json:"test_key"`
	Parse   []struct {
		Input string          `json:"input"`
		OK    json.RawMessage `json:"ok"`
		Error string          `json:"error"`
	} `json:"parse"`
	Newer []struct {
		A, B  string
		OK    *bool  `json:"ok"`
		Error string `json:"error"`
	} `json:"newer"`
	SafeURL []struct {
		Input string `json:"input"`
		Safe  bool   `json:"safe"`
	} `json:"safe_url"`
	Records []struct {
		Input string `json:"input"`
		OK    string `json:"ok"`
		Error string `json:"error"`
	} `json:"records"`
	Verify []struct {
		Raw   string `json:"raw"`
		OK    string `json:"ok"`
		Error string `json:"error"`
	} `json:"verify"`
	Scenarios []updateScenario `json:"scenarios"`
}

var (
	vectorsOnce sync.Once
	vectors     updateVectors
)

func load(t *testing.T) *updateVectors {
	vectorsOnce.Do(func() {
		data, err := os.ReadFile(filepath.Join("testdata", "generated", "vectors.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &vectors); err != nil {
			t.Fatal(err)
		}
	})
	return &vectors
}

func decode(t *testing.T, text string) any {
	t.Helper()
	value, err := pyjson.Decode([]byte(text))
	if err != nil {
		t.Fatalf("decode %q: %v", text, err)
	}
	return value
}

func compact(t *testing.T, value any) string {
	data, err := pyjson.Compact(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func testKeys(t *testing.T) map[string][]byte {
	v := load(t)
	key, _ := hex.DecodeString(v.TestKey.Public)
	return map[string][]byte{v.TestKey.Kid: key}
}

func TestVersionsMatchPython(t *testing.T) {
	v := load(t)
	for _, c := range v.Parse {
		version, ok, err := Parse(decode(t, c.Input))
		var want []int
		json.Unmarshal(c.OK, &want)
		switch {
		case c.Error != "" && err == nil:
			t.Errorf("parse(%s) = %v, want %s", c.Input, version, c.Error)
		case c.Error == "" && (err != nil || ok != (want != nil) || (ok && version != Version{want[0], want[1], want[2]})):
			t.Errorf("parse(%s) = %v %v %v, want %s", c.Input, version, ok, err, c.OK)
		}
	}
	for _, c := range v.Newer {
		newer, err := IsNewer(decode(t, c.A), decode(t, c.B))
		if (c.Error != "") != (err != nil) || (c.OK != nil && newer != *c.OK) {
			t.Errorf("is_newer(%s, %s) = %v %v, want %v %s", c.A, c.B, newer, err, c.OK, c.Error)
		}
	}
}

func TestSafeURLMatchesPython(t *testing.T) {
	for _, c := range load(t).SafeURL {
		if got := SafeURL(decode(t, c.Input)); got != c.Safe {
			t.Errorf("_safe_url(%s) = %v, want %v", c.Input, got, c.Safe)
		}
	}
}

func TestCleanRecordMatchesPython(t *testing.T) {
	for _, c := range load(t).Records {
		record, err := CleanRecord(decode(t, c.Input))
		if c.Error != "" && err == nil {
			t.Errorf("clean_record(%s) accepted, want %s", c.Input, c.Error)
		}
		if c.Error == "" && (err != nil || compact(t, record) != c.OK) {
			t.Errorf("clean_record(%s) = %v %v\nwant %s", c.Input, record, err, c.OK)
		}
	}
}

func TestVerifyMatchesPython(t *testing.T) {
	service := New(Options{Directory: t.TempDir(), Keys: testKeys(t)})
	for index, c := range load(t).Verify {
		raw, _ := base64.StdEncoding.DecodeString(c.Raw)
		record, err := service.verify(raw)
		var crash *licensing.UncaughtError
		switch {
		case c.Error == "":
			if err != nil || compact(t, record) != c.OK {
				t.Errorf("case %d: %v %v, want %s", index, record, err, c.OK)
			}
		case c.Error == "AttributeError":
			if !errors.As(err, &crash) {
				t.Errorf("case %d: %v, want an uncaught error", index, err)
			}
		case !errors.Is(err, errBadRecord):
			t.Errorf("case %d: %v, want bad_answer (%s)", index, err, c.Error)
		}
	}
}
