package autosave

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/archive"
)

func sha(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// doc builds a compact document whose diagram holds marker and whose sources are texts.
func doc(marker string, texts ...string) *pyjson.Object {
	table := pyjson.NewObject()
	ieds := []any{}
	for _, text := range texts {
		table.Set(sha(text), text)
		ieds = append(ieds, pyjson.NewObject().Set("id", sha(text)[:32]).Set("source_id", sha(text)))
	}
	return pyjson.NewObject().Set("id", "0123456789abcdef0123456789abcdef").Set("name", "P").
		Set("diagram", pyjson.NewObject().Set("marker", marker).Set("x", 1.0)).
		Set("format", int64(15)).Set("ieds", ieds).Set("scl_sources", table)
}

// derive copies top-level keys of base (sharing values) and overrides some.
func derive(base *pyjson.Object, overrides ...any) *pyjson.Object {
	next := pyjson.NewObject()
	for _, key := range base.Keys() {
		value, _ := base.Get(key)
		next.Set(key, value)
	}
	for index := 0; index+1 < len(overrides); index += 2 {
		next.Set(overrides[index].(string), overrides[index+1])
	}
	return next
}

// autosaveJSON returns the project.json of <dir>/autosave.subsim re-encoded with pyjson.
func autosaveJSON(t *testing.T, dir string) []byte {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join(dir, AutosaveName))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := archive.Decode(payload)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := pyjson.Dumps(decoded)
	return data
}

func assertAutosave(t *testing.T, dir string, want *pyjson.Object) {
	t.Helper()
	expected, _ := pyjson.Dumps(want)
	if got := autosaveJSON(t, dir); !bytes.Equal(got, expected) {
		t.Fatalf("autosave holds\n%s\nwant\n%s", got, expected)
	}
}

func saveDoc(t *testing.T, dir string, value *pyjson.Object) {
	t.Helper()
	payload, err := archive.Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteAutosave(dir, payload); err != nil {
		t.Fatal(err)
	}
}

type pyObject = pyjson.Object

// openTest opens a journal whose handle is dropped when the test ends.
func openTest(t *testing.T, dir string) (*Journal, error) {
	journal, err := OpenJournal(dir, nil)
	if journal != nil {
		t.Cleanup(func() { crash(journal) })
	}
	return journal, err
}

// crash drops the journal's handle as a dying process would, without removing the file.
func crash(j *Journal) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file != nil {
		_ = j.file.Close()
		j.file = nil
	}
}

func mustGet(object *pyjson.Object, key string) any {
	value, _ := object.Get(key)
	return value
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func journalLines(t *testing.T, dir string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, JournalName))
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, line := range bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n")) {
		lines = append(lines, string(line))
	}
	return lines
}
