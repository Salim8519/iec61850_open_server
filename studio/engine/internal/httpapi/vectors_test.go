package httpapi

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

// TestAgainstPython checks the helpers that copy Python library behaviour (Flask session
// signing, urlsplit, Werkzeug query parsing, frontend.pak, Content-Disposition) against
// the answers of the Python the engine replaces (testdata/httpapi_vectors.py).
func TestAgainstPython(t *testing.T) {
	python := os.Getenv("STUDIO_PYTHON")
	if python == "" {
		python = filepath.Join("..", "..", "..", ".venv", "Scripts", "python.exe")
	}
	if _, err := os.Stat(python); err != nil {
		t.Skipf("no CPython at %s; set STUDIO_PYTHON", python)
	}
	backend, _ := filepath.Abs(filepath.Join("..", "..", "..", "backend"))
	output, err := exec.Command(python, "-I", filepath.Join("testdata", "httpapi_vectors.py"), "--backend", backend).Output()
	if err != nil {
		t.Fatalf("vectors: %v", err)
	}
	counts := map[string]int{}
	lines := bufio.NewScanner(bytes.NewReader(output))
	lines.Buffer(make([]byte, 1<<20), 1<<24)
	for lines.Scan() {
		decoded, err := pyjson.Decode(lines.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		line := decoded.(*pyjson.Object)
		kind := text(line, "kind")
		counts[kind]++
		switch kind {
		case "cookie":
			checkCookie(t, line)
		case "netloc":
			got, err := urlNetloc(text(line, "in"))
			expectText(t, "netloc "+text(line, "in"), got, err, line)
		case "query":
			checkQuery(t, line)
		case "pak":
			checkPak(t, line)
		case "disposition":
			if got := ContentDisposition("inline", text(line, "in")); got != text(line, "out") {
				t.Errorf("disposition %q: got %q, Python %q", text(line, "in"), got, text(line, "out"))
			}
		}
	}
	if counts["cookie"] == 0 || counts["netloc"] == 0 || counts["query"] == 0 || counts["pak"] != 1 {
		t.Fatalf("missing vectors: %v", counts)
	}
	t.Logf("compared with Python: %v", counts)
}

func text(o *pyjson.Object, key string) string {
	value, _ := o.Get(key)
	s, _ := value.(string)
	return s
}

func expectText(t *testing.T, label, got string, err error, line *pyjson.Object) {
	if _, failed := line.Get("err"); failed {
		if err == nil || err.Error() != text(line, "err") {
			t.Errorf("%s: got %q, %v; Python raised %q", label, got, err, text(line, "err"))
		}
		return
	}
	if err != nil || got != text(line, "out") {
		t.Errorf("%s: got %q, %v; Python %q", label, got, err, text(line, "out"))
	}
}

func checkCookie(t *testing.T, line *pyjson.Object) {
	stamp, _ := line.Get("time")
	session, _ := line.Get("session")
	at := time.Unix(stamp.(int64), 0)
	s := newSigner(text(line, "secret"))
	got, err := s.dumps(session.(*pyjson.Object), at)
	compressed := text(line, "cookie")[0] == '.' // zlib bytes: read, not reproduced
	if !compressed && (err != nil || got != text(line, "cookie")) {
		t.Errorf("cookie at %d: got %s, Flask %s", stamp, got, text(line, "cookie"))
	}
	loaded := s.loads(text(line, "cookie"), at.Add(time.Hour))
	if value, _ := loaded.Get("authenticated"); value != true {
		t.Errorf("Flask cookie at %d not accepted", stamp)
	}
	if expired := s.loads(text(line, "cookie"), at.Add(sessionMaxAge+time.Second)); expired != nil {
		t.Errorf("cookie older than 31 days accepted")
	}
}

func checkQuery(t *testing.T, line *pyjson.Object) {
	raw, _ := hex.DecodeString(text(line, "raw"))
	r := &http.Request{URL: &url.URL{RawQuery: string(raw)}}
	value, found, err := Arg(r, "after")
	_, failed := line.Get("err")
	want, _ := line.Get("after")
	switch {
	case failed && err == nil:
		t.Errorf("query %q: Python raised, Go %q", raw, value)
	case !failed && err != nil:
		t.Errorf("query %q: Go error %v", raw, err)
	case !failed && want == nil && found:
		t.Errorf("query %q: Go found %q, Python none", raw, value)
	case !failed && want != nil && value != want:
		t.Errorf("query %q: got %q, Python %q", raw, value, want)
	}
}

func checkPak(t *testing.T, line *pyjson.Object) {
	blob, _ := hex.DecodeString(text(line, "hex"))
	files, err := unpackFrontend(blob)
	if err != nil {
		t.Fatalf("unpack: %v", err)
	}
	want, _ := line.Get("files")
	expected := want.(*pyjson.Object)
	if len(files) != expected.Len() {
		t.Fatalf("pak: %d files, Python %d", len(files), expected.Len())
	}
	for _, name := range expected.Keys() {
		if hex.EncodeToString(files[name]) != text(expected, name) {
			t.Errorf("pak file %s differs", name)
		}
	}
}
