package projectfile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"studio-engine/internal/testutil"
)

const script = "testdata/migrate.py"

// sortOfFolders are the data folders both engines must treat identically.
var sortOfFolders = map[string]func(*testing.T, string){
	"old files only": func(t *testing.T, dir string) {
		put(t, filepath.Join(dir, "autosave.subsim"), "project")
		put(t, filepath.Join(dir, "recovery", "before-ied-delete-1.subsim"), "a")
		put(t, filepath.Join(dir, "recovery", "unreadable-autosave-2.SUBSIM"), "b")
		put(t, filepath.Join(dir, "recovery", "readme.txt"), "c")
	},
	"both names": func(t *testing.T, dir string) {
		put(t, filepath.Join(dir, "autosave.subsim"), "old")
		put(t, filepath.Join(dir, "autosave.scadabench"), "new")
		put(t, filepath.Join(dir, "recovery", "x.subsim"), "older")
		put(t, filepath.Join(dir, "recovery", "x.scadabench"), "newer")
		put(t, filepath.Join(dir, "recovery", "y.subsim"), "only old")
	},
	"folders with the old name": func(t *testing.T, dir string) {
		_ = os.MkdirAll(filepath.Join(dir, "autosave.subsim"), 0o755)
		_ = os.MkdirAll(filepath.Join(dir, "recovery", "z.subsim"), 0o755)
		put(t, filepath.Join(dir, "recovery", "w.subsim"), "w")
	},
	"already migrated": func(t *testing.T, dir string) {
		put(t, filepath.Join(dir, "autosave.scadabench"), "new")
		put(t, filepath.Join(dir, "recovery", "x.scadabench"), "newer")
	},
	"empty": func(t *testing.T, dir string) {},
}

type outcome struct {
	Records []map[string]string `json:"records"`
	Journal []map[string]string `json:"journal"`
	Files   map[string]string   `json:"files"`
}

// comparable drops what differs per run or per engine: times, the run id and the OS error text.
func comparable(lines []map[string]string) []map[string]string {
	result := []map[string]string{}
	for _, line := range lines {
		kept := map[string]string{}
		for key, value := range line {
			if key != "at" && key != "correlation_id" && key != "error" {
				kept[key] = value
			}
		}
		result = append(result, kept)
	}
	return result
}

func goOutcome(t *testing.T, dir string) outcome {
	t.Helper()
	records, err := Migrate(dir)
	if err != nil {
		t.Fatal(err)
	}
	result := outcome{Files: map[string]string{}, Records: []map[string]string{}}
	for _, record := range records {
		line := map[string]string{"kind": record.Kind, "from": record.From, "to": record.To}
		if record.Reason != "" {
			line["reason"] = record.Reason
		}
		result.Records = append(result.Records, line)
	}
	result.Journal = []map[string]string{}
	if data, err := os.ReadFile(filepath.Join(dir, "logs", JournalName)); err == nil {
		for _, text := range splitLines(data) {
			line := map[string]string{}
			if err := json.Unmarshal(text, &line); err != nil {
				t.Fatalf("journal line %q: %v", text, err)
			}
			result.Journal = append(result.Journal, line)
		}
	}
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		relative, _ := filepath.Rel(dir, path)
		if err == nil && !entry.IsDir() && filepath.ToSlash(relative) != "logs/"+JournalName {
			data, _ := os.ReadFile(path)
			sum := sha256.Sum256(data)
			result.Files[filepath.ToSlash(relative)] = hex.EncodeToString(sum[:])
		}
		return nil
	})
	return result
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for index, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:index])
			start = index + 1
		}
	}
	return lines
}

// TestMigrationMatchesPython runs only with STUDIO_PYTHON set to the venv interpreter.
func TestMigrationMatchesPython(t *testing.T) {
	python := os.Getenv("STUDIO_PYTHON")
	if python == "" {
		t.Skip("set STUDIO_PYTHON to studio/.venv/Scripts/python.exe to run the live cross-check")
	}
	for name, build := range sortOfFolders {
		t.Run(name, func(t *testing.T) {
			goDir, pyDir := testutil.TempDir(t), testutil.TempDir(t)
			build(t, goDir)
			build(t, pyDir)
			out, err := exec.Command(python, "-I", script, pyDir).CombinedOutput()
			if err != nil {
				t.Fatalf("migrate.py: %v\n%s", err, out)
			}
			var fromPython outcome
			if err := json.Unmarshal(out, &fromPython); err != nil {
				t.Fatalf("migrate.py output: %v\n%s", err, out)
			}
			fromGo := goOutcome(t, goDir)
			if !reflect.DeepEqual(comparable(fromGo.Records), comparable(fromPython.Records)) ||
				!reflect.DeepEqual(comparable(fromGo.Journal), comparable(fromPython.Journal)) ||
				!reflect.DeepEqual(fromGo.Files, fromPython.Files) {
				t.Fatalf("Go and Python differ\nGo: %+v\nPython: %+v", fromGo, fromPython)
			}
		})
	}
}
