package jsonfile

import (
	"os"
	"path/filepath"
	"studio-engine/internal/testutil"
	"testing"

	"studio-engine/internal/pyjson"
)

func TestEncodeIsIndentTwoWithCRLF(t *testing.T) {
	value := pyjson.NewObject().Set("tour_seen_version", int64(1)).Set("empty", pyjson.NewObject()).
		Set("list", []any{1.0, "\u00e9", []any{}, nil})
	got, err := Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\r\n  \"tour_seen_version\": 1,\r\n  \"empty\": {},\r\n  \"list\": [\r\n    1.0,\r\n" +
		"    \"\\u00e9\",\r\n    [],\r\n    null\r\n  ]\r\n}"
	if string(got) != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestWriteAndReadTolerance(t *testing.T) {
	dir := testutil.TempDir(t)
	path := filepath.Join(dir, "nested", "preferences.json")
	if err := Write(path, pyjson.NewObject().Set("a", true)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Fatal("temp file left behind")
	}
	value, ok := Read(path)
	if object, isObject := value.(*pyjson.Object); !ok || !isObject || object.Len() != 1 {
		t.Fatalf("read back %v %v", value, ok)
	}
	for name, content := range map[string]string{
		"bom":     "\xef\xbb\xbf{}",
		"utf8":    "{\"a\": \"\xff\"}",
		"surr":    "{\"a\": \"\xed\xa0\x80\"}",
		"invalid": "{",
	} {
		file := filepath.Join(dir, name)
		_ = os.WriteFile(file, []byte(content), 0o666)
		if _, ok := Read(file); ok {
			t.Errorf("%s accepted", name)
		}
	}
	lf := filepath.Join(dir, "lf")
	_ = os.WriteFile(lf, []byte("{\n\r  \"a\": 1\r}"), 0o666)
	if _, ok := Read(lf); !ok {
		t.Error("line ends refused")
	}
	if _, ok := Read(filepath.Join(dir, "missing")); ok {
		t.Error("missing file accepted")
	}
}
