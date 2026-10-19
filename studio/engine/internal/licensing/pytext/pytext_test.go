package pytext

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unicode"

	"studio-engine/internal/pyjson"
)

type vectors struct {
	Space  []rune   `json:"space"`
	Digits [][2]int `json:"digits"`
	Ints   []struct {
		Text  string  `json:"text"`
		Value *string `json:"value"`
	} `json:"ints"`
	Indent []struct{ Input, Output string } `json:"indent"`
	Loads  []struct {
		Data  string  `json:"data"`
		Bytes *string `json:"bytes"`
		Str   *string `json:"str"`
	} `json:"loads"`
	WriteText []struct{ Text, Disk string } `json:"write_text"`
}

func load(t *testing.T) vectors {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "generated", "vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v vectors
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestWhitespaceAndDigitsMatchCPython(t *testing.T) {
	v := load(t)
	space := map[rune]bool{}
	for _, r := range v.Space {
		space[r] = true
	}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if IsSpace(r) != space[r] {
			t.Fatalf("IsSpace(%U) = %v", r, IsSpace(r))
		}
	}
	digits := map[rune]int{}
	for _, pair := range v.Digits {
		digits[rune(pair[0])] = pair[1]
	}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		want, isDigit := digits[r]
		got, ok := DigitValue(r)
		if ok != isDigit || got != want {
			t.Fatalf("DigitValue(%U) = %d %v, want %d %v", r, got, ok, want, isDigit)
		}
	}
}

func TestParseIntMatchesCPython(t *testing.T) {
	for _, c := range load(t).Ints {
		value, err := ParseInt(c.Text)
		switch {
		case c.Value == nil && err == nil:
			t.Errorf("int(%q) = %s, want ValueError", c.Text, value)
		case c.Value != nil && (err != nil || value.String() != *c.Value):
			t.Errorf("int(%q) = %v %v, want %s", c.Text, value, err, *c.Value)
		}
	}
}

func TestDumpsIndentMatchesCPython(t *testing.T) {
	for _, c := range load(t).Indent {
		value, err := pyjson.Decode([]byte(c.Input))
		if err != nil {
			t.Fatalf("decode %s: %v", c.Input, err)
		}
		got, err := DumpsIndent(value)
		if err != nil || string(got) != c.Output {
			t.Errorf("DumpsIndent(%s)\n got %q\nwant %q", c.Input, got, c.Output)
		}
	}
}

func compact(value any, err error) *string {
	if err != nil {
		return nil
	}
	out, _ := pyjson.Compact(value)
	text := string(out)
	return &text
}

func same(a, b *string) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }

func TestLoadsMatchesCPython(t *testing.T) {
	for _, c := range load(t).Loads {
		data, _ := base64.StdEncoding.DecodeString(c.Data)
		if got := compact(LoadsBytes(data)); !same(got, c.Bytes) {
			t.Errorf("json.loads(bytes %q) mismatch", data)
		}
		text, err := ReadText(data, nil)
		var got *string
		if err == nil {
			got = compact(LoadsString(text))
		}
		if !same(got, c.Str) {
			t.Errorf("json.loads(str %q) mismatch", data)
		}
	}
}

func TestWriteReplaceMatchesWriteText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "file.json")
	for _, c := range load(t).WriteText {
		if err := WriteReplace(path, []byte(c.Text)); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(path)
		want, _ := base64.StdEncoding.DecodeString(c.Disk)
		if runtime.GOOS != "windows" {
			want = []byte(c.Text) // the vectors were written on Windows
		}
		if string(got) != string(want) {
			t.Errorf("write_text(%q) = %q, want %q", c.Text, got, want)
		}
		if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
			t.Errorf("temporary file left behind")
		}
	}
	if _, ok := ReadJSON(filepath.Join(t.TempDir(), "missing.json")); ok {
		t.Error("a missing file must fall back to the default")
	}
}

func TestStripAndRemoveSpace(t *testing.T) {
	if got := Strip("　 a b\x1c"); got != "a b" {
		t.Errorf("Strip = %q", got)
	}
	if got := RemoveSpace(" SUBS- AB\tCD\n"); got != "SUBS-ABCD" {
		t.Errorf("RemoveSpace = %q", got)
	}
}
