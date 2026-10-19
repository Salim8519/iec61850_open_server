package pyjson

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

// TestAgainstCPython decodes documents written by CPython and must re-encode them to the
// same bytes as Flask's jsonify and json.dumps (ADR-0075 gate). PYJSON_CASES sets the count
// (the P0 gate runs 1,000,000); STUDIO_PYTHON points to a Python with Flask installed.
func TestAgainstCPython(t *testing.T) {
	python := os.Getenv("STUDIO_PYTHON")
	if python == "" {
		python = filepath.Join("..", "..", "..", ".venv", "Scripts", "python.exe")
	}
	if _, err := os.Stat(python); err != nil {
		t.Skipf("no Python with Flask at %s; set STUDIO_PYTHON", python)
	}
	cases := 20000
	if text := os.Getenv("PYJSON_CASES"); text != "" {
		cases, _ = strconv.Atoi(text)
	}
	seed := os.Getenv("PYJSON_SEED")
	if seed == "" {
		seed = "61850"
	}
	command := exec.Command(python, "-I", filepath.Join("testdata", "pyjson_vectors.py"),
		"--seed", seed, "--count", strconv.Itoa(cases))
	command.Stderr = os.Stderr
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	lines := bufio.NewScanner(output)
	lines.Buffer(make([]byte, 1<<20), 1<<26)
	checked, failures := 0, 0
	for lines.Scan() {
		input := append([]byte(nil), lines.Bytes()...)
		if !lines.Scan() {
			break
		}
		flask := lines.Text()
		if !lines.Scan() {
			break
		}
		dumps := lines.Text()
		checked++
		value, err := Decode(input)
		if err != nil {
			t.Errorf("case %d: decode %s: %v", checked, input, err)
			failures++
		} else {
			if got, _ := Flask(value); string(got) != flask+"\n" {
				t.Errorf("case %d Flask\n  in  %s\n got %s\nwant %s", checked, input, got, flask)
				failures++
			}
			if got, _ := Dumps(value); string(got) != dumps {
				t.Errorf("case %d dumps\n  in  %s\n got %s\nwant %s", checked, input, got, dumps)
				failures++
			}
			if got, _ := Compact(value); string(got) != string(input) {
				t.Errorf("case %d compact round trip\n got %s\nwant %s", checked, got, input)
				failures++
			}
		}
		if failures >= 20 {
			break
		}
	}
	_ = command.Process.Kill()
	_ = command.Wait()
	if failures == 0 && checked != cases {
		t.Fatalf("Python wrote %d of %d documents", checked, cases)
	}
	t.Logf("%d documents compared with CPython/Flask, %d differences", checked, failures)
}
