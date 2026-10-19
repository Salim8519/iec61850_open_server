package pycompat

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"studio-engine/internal/pyjson"
)

// TestAgainstCPython compares every rule with CPython's own answers (testdata/
// pycompat_vectors.py): full code point tables plus PYCOMPAT_CASES random cases (default
// 20000). STUDIO_PYTHON points to the Python the Python engine runs on.
func TestAgainstCPython(t *testing.T) {
	python := os.Getenv("STUDIO_PYTHON")
	if python == "" {
		python = filepath.Join("..", "..", "..", ".venv", "Scripts", "python.exe")
	}
	if _, err := os.Stat(python); err != nil {
		t.Skipf("no CPython at %s; set STUDIO_PYTHON", python)
	}
	cases := 20000
	if text := os.Getenv("PYCOMPAT_CASES"); text != "" {
		cases, _ = strconv.Atoi(text)
	}
	seed := os.Getenv("PYCOMPAT_SEED")
	if seed == "" {
		seed = "61850"
	}
	command := exec.Command(python, "-I", filepath.Join("testdata", "pycompat_vectors.py"),
		"--seed", seed, "--count", strconv.Itoa(cases))
	command.Stderr = os.Stderr
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = command.Process.Kill(); _ = command.Wait() }()
	lines := bufio.NewScanner(output)
	lines.Buffer(make([]byte, 1<<20), 1<<26)
	counts := map[string]int{}
	failures := 0
	for lines.Scan() && failures < 30 {
		value, err := pyjson.Decode(lines.Bytes())
		if err != nil {
			t.Fatalf("vector line: %v", err)
		}
		c := vector{t: t, o: value.(*pyjson.Object)}
		kind := c.str("f")
		counts[kind]++
		if !checkVector(kind, c) {
			failures++
		}
	}
	total := 0
	for _, n := range counts {
		total += n
	}
	if failures == 0 && total != cases+4 {
		t.Fatalf("CPython wrote %d of %d vectors", total, cases+4)
	}
	t.Logf("compared with CPython: %v, %d differences", counts, failures)
}

func checkVector(kind string, c vector) bool {
	switch kind {
	case "table":
		return checkTable(c)
	case "int":
		return checkInt(c)
	case "round":
		return checkRound(c)
	case "fromisoformat":
		return checkFromIsoformat(c)
	case "isoformat":
		return checkIsoformat(c)
	case "fromtimestamp":
		return checkFromTimestamp(c)
	case "utf8":
		return checkUTF8(c)
	case "text":
		return checkText(c)
	}
	c.t.Errorf("unknown vector kind %q", kind)
	return false
}

// vector reads one CPython answer.
type vector struct {
	t *testing.T
	o *pyjson.Object
}

func (c vector) get(key string) any {
	value, _ := c.o.Get(key)
	return value
}

func (c vector) str(key string) string {
	text, _ := c.get(key).(string)
	return text
}

func (c vector) number(key string) float64 {
	switch value := c.get(key).(type) {
	case float64:
		return value
	case int64:
		return float64(value)
	}
	return 0
}

// expect compares a Go result or error with the vector's "out" or "err".
func (c vector) expect(label, got string, err error) bool {
	if want, failed := c.o.Get("err"); failed {
		if err == nil || err.Error() != want.(string) {
			c.t.Errorf("%s: got %q (err %v), CPython raised %s: %v", label, got, err, c.str("cls"), want)
			return false
		}
		return true
	}
	if err != nil || got != c.str("out") {
		c.t.Errorf("%s: got %q (err %v), CPython %q", label, got, err, c.str("out"))
		return false
	}
	return true
}
