package crosscheck

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/archive"
	"studio-engine/internal/store/autosave"
)

const script = "../testdata/crosscheck.py"

func digestOf(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// compactDoc builds a format-15 document that Python's expand() and compact() reproduce.
func compactDoc(label string, texts ...string) *pyjson.Object {
	huge, _ := new(big.Int).SetString("1180591620717411303424", 10)
	node := pyjson.NewObject().Set("id", "n1").Set("position", pyjson.NewObject().Set("x", 360.0).
		Set("y", math.Copysign(0, -1))).Set("data", pyjson.NewObject().Set("label", label).
		Set("values", []any{1e16, 1e-05, 0.1 + 0.2, huge, nil, true}))
	table := pyjson.NewObject()
	ieds := []any{}
	for index, text := range texts {
		ieds = append(ieds, pyjson.NewObject().Set("id", fmt.Sprintf("%032x", index+1)).
			Set("name", fmt.Sprint("IED", index+1)).Set("filename", "a.scd").Set("port", int64(102)).
			Set("source_id", digestOf(text)))
		table.Set(digestOf(text), text)
	}
	return pyjson.NewObject().Set("id", "0123456789abcdef0123456789abcdef").Set("name", label).
		Set("diagram", pyjson.NewObject().Set("nodes", []any{node}).Set("edges", []any{})).
		Set("sources", []any{}).Set("format", int64(15)).Set("ieds", ieds).Set("scl_sources", table)
}

func publicTexts(t *testing.T) []string {
	root := filepath.Join("..", "..", "..", "..", "..")
	names, _ := filepath.Glob(filepath.Join(root, "scd", "*.scd"))
	names = append(names, filepath.Join(root, "marketing", "videos", "demo", "scadabench_demo.scd"))
	var texts []string
	for _, name := range names {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("public SCL: %v", err)
		}
		texts = append(texts, strings.TrimPrefix(string(data), "\xef\xbb\xbf"))
	}
	return texts
}

// writeGoArchives writes <name>.subsim and the expected <name>.json for the verifier.
func writeGoArchives(t *testing.T, dir string) {
	write := func(name string, payload []byte, doc *pyjson.Object) {
		expected, _ := pyjson.Dumps(doc)
		_ = os.WriteFile(filepath.Join(dir, name+".subsim"), payload, 0o666)
		_ = os.WriteFile(filepath.Join(dir, name+".json"), expected, 0o666)
	}
	encoder := archive.NewEncoder()
	big := strings.Repeat("<IED name=\"x\" desc=\"é\U0001F600\"/>\r\n\t", 90000) // > 3 chunks
	docs := map[string]*pyjson.Object{
		"small":   compactDoc("Démo <&> \x7f", "<SCL>☃</SCL>", "<SCL/>"),
		"public":  compactDoc("public", publicTexts(t)...),
		"chunked": compactDoc("chunked", big, "<SCL>two</SCL>"),
		"edited":  compactDoc("edited after a cached save", big, "<SCL>two</SCL>"),
		"added":   compactDoc("source added", big, "<SCL>two</SCL>", "<SCL>three</SCL>"),
		"removed": compactDoc("source removed", "<SCL>two</SCL>"),
		"empty":   compactDoc("no IEDs"),
	}
	for _, name := range []string{"small", "public", "chunked", "edited", "added", "removed", "empty"} {
		payload, err := encoder.Encode(docs[name]) // later ones reuse cached segments
		if err != nil {
			t.Fatal(err)
		}
		write(name, payload, docs[name])
	}
	// An autosave produced by folding a journal.
	foldDir := t.TempDir()
	journal, _ := autosave.OpenJournal(foldDir, nil)
	if err := journal.Commit(1, nil, docs["small"]); err != nil {
		t.Fatal(err)
	}
	if err := journal.Commit(2, docs["small"], docs["added"]); err != nil {
		t.Fatal(err)
	}
	_ = journal.Close() // records remain, as after a crash
	if _, err := autosave.Fold(foldDir, nil); err != nil {
		t.Fatal(err)
	}
	folded, _ := os.ReadFile(filepath.Join(foldDir, autosave.AutosaveName))
	write("folded", folded, docs["added"])
}

func runPython(t *testing.T, python string, args ...string) []byte {
	t.Helper()
	command := exec.Command(python, append([]string{"-I", script}, args...)...)
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("crosscheck.py %v: %v\n%s", args, err, out)
	}
	return out
}

// TestPythonCrossCheck runs only with STUDIO_PYTHON set to the venv interpreter.
func TestPythonCrossCheck(t *testing.T) {
	python := os.Getenv("STUDIO_PYTHON")
	if python == "" {
		t.Skip("set STUDIO_PYTHON to studio/.venv/Scripts/python.exe to run the live cross-check")
	}
	goDir := t.TempDir()
	writeGoArchives(t, goDir)
	var report struct {
		Files  int      `json:"files"`
		Errors []string `json:"errors"`
	}
	out := runPython(t, python, "verify", goDir)
	if err := json.Unmarshal(bytes.TrimSpace(out), &report); err != nil || report.Files != 8 || len(report.Errors) > 0 {
		t.Fatalf("Python verification: %s", out)
	}
	generated := t.TempDir()
	runPython(t, python, "generate", generated)
	err := filepath.WalkDir(fixtures, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Name() == "python_archive.subsim" {
			return err
		}
		relative, _ := filepath.Rel(fixtures, path)
		committed, _ := os.ReadFile(path)
		fresh, readErr := os.ReadFile(filepath.Join(generated, relative))
		if readErr != nil || !bytes.Equal(committed, fresh) {
			t.Errorf("golden %s is out of date with Python", relative)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
