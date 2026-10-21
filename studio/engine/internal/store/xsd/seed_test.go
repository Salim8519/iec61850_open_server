package xsd

import (
	"os"
	"path/filepath"
	"reflect"
	"studio-engine/internal/testutil"
	"testing"

	"studio-engine/internal/scl"
)

// goldenSeed holds the verdicts Python (lxml 6.0.2) produced for the SCL corpus.
var goldenSeed = filepath.Join("..", "..", "scl", "testdata", "golden", "xsd_seed.json")

func TestGoldenSeedIsServedWithoutHelper(t *testing.T) {
	data, err := os.ReadFile(goldenSeed)
	if err != nil {
		t.Skipf("golden seed not available: %v", err)
	}
	entries, err := DecodeSeed(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("the golden seed has no verdicts")
	}
	dataDir := testutil.TempDir(t)
	config := Config{HelperPath: filepath.Join(dataDir, "missing", "studio_xsd.exe"),
		SchemasDir: bundledSchemas, DataDir: dataDir}
	seeding := New(config)
	stored, err := seeding.ImportSeed(entries)
	if err != nil || stored != len(entries) {
		t.Fatalf("imported %d of %d: %v", stored, len(entries), err)
	}
	seeding.Close()

	// A fresh validator (new process) finds every verdict on disk; the helper path does
	// not exist, so any cache miss would fail with the missing-helper error.
	validator := New(config)
	defer validator.Close()
	invalid := 0
	for index, entry := range entries {
		verdict, err := validator.lookup(entry.SHA256, entry.Schema)
		if err != nil {
			t.Fatalf("entry %d (%s): %v", index, entry.SHA256, err)
		}
		if !reflect.DeepEqual(verdict, entry.Verdict) {
			t.Fatalf("entry %d: got %+v, want %+v", index, verdict, entry.Verdict)
		}
		if verdict.Status == "invalid" {
			invalid++
		}
	}
	if validator.HelperStarts() != 0 {
		t.Fatalf("seeded verdicts must never start the helper, started %d", validator.HelperStarts())
	}
	t.Logf("%d seeded verdicts served from the cache (%d invalid), helper never started", len(entries), invalid)
}

func TestSeededDocumentValidatesFromCache(t *testing.T) {
	source := []byte("<SCL>seeded</SCL>")
	verdict := scl.SchemaVerdict{Status: "invalid", Errors: []scl.SchemaError{{Line: 1, Message: "from Python"}}}
	dataDir := testutil.TempDir(t)
	validator := New(Config{HelperPath: filepath.Join(dataDir, "none.exe"), SchemasDir: bundledSchemas, DataDir: dataDir})
	defer validator.Close()
	entry := SeedEntry{SHA256: sha256Hex(source), Schema: "SCL2007B", Verdict: verdict}
	if _, err := validator.ImportSeed([]SeedEntry{entry}); err != nil {
		t.Fatal(err)
	}
	got, err := validator.Validate(source, "SCL2007B")
	if err != nil || !reflect.DeepEqual(got, verdict) {
		t.Fatalf("got %+v %v", got, err)
	}
	if _, err := validator.Validate(source, "SCL2003"); err == nil {
		t.Fatal("another schema is another key and must miss")
	} else {
		requireRuntime(t, err, "studio_xsd helper is missing")
	}
}

func TestDecodeSeedChecksEntries(t *testing.T) {
	hash := sha256Hex([]byte("x"))
	good := `[{"sha256": "` + hash + `", "schema": "SCL2003", "status": "valid", "errors": []},` +
		`{"sha256": null, "schema": null, "status": "unsupported", "errors": []}]`
	entries, err := DecodeSeed([]byte(good))
	if err != nil || len(entries) != 1 || entries[0].Verdict.Status != "valid" {
		t.Fatalf("got %+v %v", entries, err)
	}
	for name, text := range map[string]string{
		"not a list":   `{"entries": []}`,
		"bad hash":     `[{"sha256": "ABC", "schema": "SCL2003", "status": "valid", "errors": []}]`,
		"bad schema":   `[{"sha256": "` + hash + `", "schema": "SCL2009", "status": "valid", "errors": []}]`,
		"bad errors":   `[{"sha256": "` + hash + `", "schema": "SCL2003", "status": "invalid", "errors": [1]}]`,
		"invalid json": `[`,
	} {
		if _, err := DecodeSeed([]byte(text)); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

// lookup is Validate for a document known only by its hash (seed checks).
func (v *Validator) lookup(sourceSHA256, schema string) (scl.SchemaVerdict, error) {
	key, err := v.key(sourceSHA256, schema)
	if err != nil {
		return scl.SchemaVerdict{}, err
	}
	if verdict, found := v.cache.get(key); found {
		return verdict, nil
	}
	_, err = v.client.running()
	return scl.SchemaVerdict{}, err
}
