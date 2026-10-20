package desktop

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// browserCandidates is app_window._CANDIDATES: Edge, then Chrome, in this order.
var browserCandidates = [...]struct{ variable, relative string }{
	{"ProgramFiles(x86)", "Microsoft/Edge/Application/msedge.exe"},
	{"ProgramFiles", "Microsoft/Edge/Application/msedge.exe"},
	{"ProgramFiles", "Google/Chrome/Application/chrome.exe"},
	{"ProgramFiles(x86)", "Google/Chrome/Application/chrome.exe"},
	{"LOCALAPPDATA", "Google/Chrome/Application/chrome.exe"},
}

// profileNames are the window's own browser profiles inside the data folder.
var profileNames = [...]string{"browser-profile", "browser-profile-2", "browser-profile-3"}

// FindBrowser is app_window.find_browser: the first candidate whose environment variable
// is set (looked up with getenv, normally os.Getenv) and whose executable is a regular
// file; "" when there is none.
func FindBrowser(getenv func(string) string) string {
	for _, candidate := range browserCandidates {
		base := getenv(candidate.variable)
		if base == "" {
			continue
		}
		path := filepath.Join(base, filepath.FromSlash(candidate.relative))
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}

// AppWindowCommand is app_window.app_window_command: a chromeless --app window with its
// own profile, no first-run or default-browser prompts, 1440x900.
func AppWindowCommand(browser, url, profile string) []string {
	return []string{browser, "--app=" + url, "--user-data-dir=" + profile, "--no-first-run",
		"--no-default-browser-check", "--window-size=1440,900"}
}

// ProfileInUse is app_window.profile_in_use: Chromium keeps <profile>/lockfile open while
// a browser runs on that profile, so deleting it fails. A missing lockfile means free; a
// stale one is deleted (free); any other failure means in use.
func ProfileInUse(profile string) bool {
	err := deleteFile(filepath.Join(profile, "lockfile"))
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return true
}

// ChooseProfile is app_window.choose_profile: the first of browser-profile,
// browser-profile-2, browser-profile-3 in dataDir that can be created and is not in use,
// so a leftover window from an earlier run never adopts ours. If a profile folder cannot
// be created, or all three are busy, a new temporary ScadaBench-browser-* folder is used.
func ChooseProfile(dataDir string) (string, error) {
	for _, name := range profileNames {
		profile := filepath.Join(dataDir, name)
		if err := os.MkdirAll(profile, 0o755); err != nil {
			break
		}
		if !ProfileInUse(profile) {
			return profile, nil
		}
	}
	return os.MkdirTemp("", "ScadaBench-browser-")
}
