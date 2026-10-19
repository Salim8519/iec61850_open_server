//go:build windows

package netcfg

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// studioPython finds a Python with the studio dependencies: STUDIO_PYTHON, else the first
// studio/.venv up the folder tree (a worktree sits inside the main checkout).
func studioPython(t *testing.T) string {
	if python := os.Getenv("STUDIO_PYTHON"); python != "" {
		return python
	}
	folder, _ := filepath.Abs(".")
	for {
		candidate := filepath.Join(folder, "studio", ".venv", "Scripts", "python.exe")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(folder)
		if parent == folder {
			t.Skip("no studio Python found; set STUDIO_PYTHON")
		}
		folder = parent
	}
}

// pythonListing runs the frozen ip_helper/adapters code. It only reads the adapter list.
const pythonListing = `import json, sys
sys.path.insert(0, sys.argv[1])
from studio_app.networking import adapters, ip_helper
raw = ip_helper.adapters()
print(json.dumps(raw))
print(json.dumps(adapters.normalize(raw)))
`

func goListing(t *testing.T) (string, string) {
	raw, err := RawAdapters()
	if err != nil {
		t.Fatal(err)
	}
	return dumpsText(t, raw), dumpsText(t, AdaptersObject(Normalize(raw)))
}

// TestAdapterListingMatchesPython lists the real adapters in Go and in Python (read-only
// IP Helper calls on both sides) and requires the same raw records and normalized list.
func TestAdapterListingMatchesPython(t *testing.T) {
	python := studioPython(t)
	backend, _ := filepath.Abs(filepath.Join("..", "..", "..", "backend"))
	firstRaw, firstList := goListing(t)
	output, err := exec.Command(python, "-I", "-B", "-c", pythonListing, backend).Output()
	if err != nil {
		t.Fatalf("python listing: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(output), "\r\n", "\n")), "\n")
	if len(lines) != 2 {
		t.Fatalf("python printed %q", output)
	}
	secondRaw, secondList := goListing(t)
	// A link speed may change between the calls; Python must equal one of the Go listings.
	if lines[0] != firstRaw && lines[0] != secondRaw {
		t.Errorf("raw adapters differ:\n go %s\npy %s", firstRaw, lines[0])
	}
	if lines[1] != firstList && lines[1] != secondList {
		t.Errorf("normalized adapters differ:\n go %s\npy %s", firstList, lines[1])
	}
	t.Logf("%d adapters listed identically", strings.Count(firstList, `"guid"`))
}
