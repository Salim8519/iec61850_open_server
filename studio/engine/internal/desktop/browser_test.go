package desktop

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"studio-engine/internal/testutil"
)

func TestAppWindowCommand(t *testing.T) {
	got := AppWindowCommand(`C:\Edge\msedge.exe`, "http://127.0.0.1:8765/#session=x&window=app", `D:\data\browser-profile`)
	want := []string{`C:\Edge\msedge.exe`, "--app=http://127.0.0.1:8765/#session=x&window=app",
		`--user-data-dir=D:\data\browser-profile`, "--no-first-run", "--no-default-browser-check",
		"--window-size=1440,900"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command\n got %q\nwant %q", got, want)
	}
}

func makeExecutable(t *testing.T, base, relative string) string {
	t.Helper()
	path := filepath.Join(base, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("exe"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFindBrowserCandidateOrder(t *testing.T) {
	root := testutil.TempDir(t)
	env := map[string]string{
		"ProgramFiles(x86)": filepath.Join(root, "x86"),
		"ProgramFiles":      filepath.Join(root, "pf"),
		"LOCALAPPDATA":      filepath.Join(root, "local"),
	}
	getenv := func(name string) string { return env[name] }
	if got := FindBrowser(getenv); got != "" {
		t.Fatalf("nothing installed, got %q", got)
	}
	// Installed in reverse preference order; each step must switch to the better one.
	steps := []struct{ variable, relative string }{
		{"LOCALAPPDATA", "Google/Chrome/Application/chrome.exe"},
		{"ProgramFiles(x86)", "Google/Chrome/Application/chrome.exe"},
		{"ProgramFiles", "Google/Chrome/Application/chrome.exe"},
		{"ProgramFiles", "Microsoft/Edge/Application/msedge.exe"},
		{"ProgramFiles(x86)", "Microsoft/Edge/Application/msedge.exe"},
	}
	for _, step := range steps {
		want := makeExecutable(t, env[step.variable], step.relative)
		if got := FindBrowser(getenv); got != want {
			t.Fatalf("after installing %s: got %q want %q", want, got, want)
		}
	}
	delete(env, "ProgramFiles(x86)")
	if got := FindBrowser(getenv); !strings.HasSuffix(got, filepath.Join("pf", "Microsoft", "Edge", "Application", "msedge.exe")) {
		t.Fatalf("unset variable is skipped, got %q", got)
	}
	// A directory with the executable's name is not a browser.
	only := filepath.Join(root, "dironly")
	if err := os.MkdirAll(filepath.Join(only, "Google", "Chrome", "Application", "chrome.exe"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := FindBrowser(func(name string) string {
		if name == "LOCALAPPDATA" {
			return only
		}
		return ""
	}); got != "" {
		t.Fatalf("directory accepted as browser: %q", got)
	}
}

func TestProfileInUse(t *testing.T) {
	profile := testutil.TempDir(t)
	if ProfileInUse(profile) {
		t.Fatal("no lockfile: free")
	}
	lock := filepath.Join(profile, "lockfile")
	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if ProfileInUse(profile) {
		t.Fatal("stale lockfile: free")
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Fatal("stale lockfile must be deleted")
	}
	if err := os.MkdirAll(filepath.Join(lock, "held"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !ProfileInUse(profile) {
		t.Fatal("undeletable lockfile (a folder) means in use")
	}
	if ProfileInUse(filepath.Join(profile, "missing")) {
		t.Fatal("missing profile folder: free")
	}
}

func TestChooseProfile(t *testing.T) {
	data := testutil.TempDir(t)
	got, err := ChooseProfile(data)
	if err != nil || got != filepath.Join(data, "browser-profile") {
		t.Fatalf("first choice %q %v", got, err)
	}
	busy := func(name string) {
		if err := os.MkdirAll(filepath.Join(data, name, "lockfile", "held"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	busy("browser-profile")
	if got, _ = ChooseProfile(data); got != filepath.Join(data, "browser-profile-2") {
		t.Fatalf("first busy: %q", got)
	}
	busy("browser-profile-2")
	busy("browser-profile-3")
	got, err = ChooseProfile(data)
	if err != nil || !strings.HasPrefix(filepath.Base(got), "ScadaBench-browser-") || filepath.Dir(got) == data {
		t.Fatalf("all busy: %q %v", got, err)
	}
	_ = os.RemoveAll(got)
	file := filepath.Join(data, "a-file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = ChooseProfile(file)
	if err != nil || !strings.HasPrefix(filepath.Base(got), "ScadaBench-browser-") {
		t.Fatalf("uncreatable profile: %q %v", got, err)
	}
	_ = os.RemoveAll(got)
}
