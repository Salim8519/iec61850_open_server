package launcher

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func probe(t *testing.T, args ...string) *exec.Cmd {
	python := os.Getenv("STUDIO_PYTHON")
	if python == "" {
		python = filepath.Join("..", "..", "..", ".venv", "Scripts", "python.exe")
	}
	if _, err := os.Stat(python); err != nil || runtime.GOOS != "windows" {
		t.Skipf("needs Windows and the studio Python (STUDIO_PYTHON)")
	}
	backend, _ := filepath.Abs(filepath.Join("..", "..", "..", "backend"))
	return exec.Command(python, append([]string{"-I", filepath.Join("testdata", "lease_probe.py"), backend}, args...)...)
}

func probeOutput(t *testing.T, args ...string) string {
	output, err := probe(t, args...).Output()
	if err != nil {
		t.Fatalf("probe %v: %v", args, err)
	}
	return strings.TrimSpace(string(output))
}

// TestLeaseSharedWithPython: the Go and Python engines exclude each other on one data
// folder through the same byte lock, and release it for each other.
func TestLeaseSharedWithPython(t *testing.T) {
	folder := t.TempDir()
	lease, err := AcquireLease(folder)
	if err != nil {
		t.Fatal(err)
	}
	if got := probeOutput(t, folder, "try"); got != LeaseBusyText {
		t.Errorf("Python while Go holds the lease: %q", got)
	}
	lease.Release()
	if got := probeOutput(t, folder, "try"); got != "ok" {
		t.Errorf("Python after Go released: %q", got)
	}
	holder := probe(t, folder, "hold")
	stdin, _ := holder.StdinPipe()
	stdout, _ := holder.StdoutPipe()
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	if line, _ := bufio.NewReader(stdout).ReadString('\n'); strings.TrimSpace(line) != "locked" {
		t.Fatalf("Python holder: %q", line)
	}
	if _, err := AcquireLease(folder); err == nil || err.Error() != LeaseBusyText {
		t.Errorf("Go while Python holds the lease: %v", err)
	}
	_ = stdin.Close()
	_ = holder.Wait()
	again, err := AcquireLease(folder)
	if err != nil {
		t.Fatalf("Go after Python released: %v", err)
	}
	again.Release()
	if data, _ := os.ReadFile(filepath.Join(folder, "studio.lock")); string(data) != "0" {
		t.Errorf("studio.lock content %q", data)
	}
}

func TestRecordLikePython(t *testing.T) {
	for _, pid := range []int{0, 4242} {
		pythonFolder, goFolder := t.TempDir(), t.TempDir()
		probeOutput(t, pythonFolder, "record", "http://127.0.0.1:8765", map[bool]string{true: "4242", false: "0"}[pid != 0])
		if err := WriteRecord(goFolder, "http://127.0.0.1:8765", pid); err != nil {
			t.Fatal(err)
		}
		want, _ := os.ReadFile(filepath.Join(pythonFolder, RecordName))
		got, _ := os.ReadFile(filepath.Join(goFolder, RecordName))
		if string(got) != string(want) {
			t.Errorf("launcher.json pid %d: got %q, Python %q", pid, got, want)
		}
		if ReadWindowPID(pythonFolder) != pid {
			t.Errorf("window pid read back: %d", ReadWindowPID(pythonFolder))
		}
	}
}

func TestOptionsLikeArgparse(t *testing.T) {
	t.Setenv("STUDIO_DEV_TOKEN", "")
	options, err := ParseOptions([]string{"--dev", "--port=+20001", "--data-dir", `C:\d`, "--front", "http://x/"})
	if err != nil || !options.Dev || options.Port != 20001 || options.DataDir != `C:\d` ||
		options.FrontendURL != "http://x/" || options.DevToken != "" {
		t.Fatalf("parse: %+v %v", options, err)
	}
	for args, want := range map[string]string{
		"--port abc":   "argument --port: invalid int value: 'abc'",
		"--d x":        "ambiguous option: --d could match --data-dir, --dev, --dev-token",
		"--bogus":      "unrecognized arguments: --bogus",
		"--port":       "argument --port: expected one argument",
		"--dev=1":      "argument --dev: ignored explicit argument '1'",
		"--no-browser": "",
	} {
		_, err := ParseOptions(strings.Fields(args))
		if (err == nil && want != "") || (err != nil && err.Error() != want) {
			t.Errorf("%s: %v; want %q", args, err, want)
		}
	}
}

func TestEngineChoice(t *testing.T) {
	t.Setenv(EngineVariable, "")
	if engine, _ := ResolveEngine(""); engine != EngineGo {
		t.Errorf("default engine %q", engine)
	}
	t.Setenv(EngineVariable, "Python")
	if engine, _ := ResolveEngine(""); engine != EnginePython {
		t.Errorf("environment engine %q", engine)
	}
	if engine, _ := ResolveEngine("go"); engine != EngineGo {
		t.Errorf("flag beats environment: %q", engine)
	}
	if _, err := ResolveEngine("rust"); err == nil {
		t.Error("unknown engine accepted")
	}
}
