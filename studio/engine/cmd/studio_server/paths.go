package main

import (
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"studio-engine/internal/launcher"
)

type paths struct {
	data     string
	frontend string
	native   string // folder of studio_ied.exe and the other native programs
	schemas  string // folder holding SCL2003, SCL2007B and SCL2007B4
	version  string
}

// version is set by the packaged build (-ldflags "-X main.version=0.1.15"); development
// reads studio_app/version.py, the single source of the version.
var version = ""

// resolvePaths mirrors Settings.defaults: --data-dir wins; otherwise a "data" folder beside
// the executable, else %LOCALAPPDATA%\ScadaBench\data. The frontend is frontend.pak or the
// frontend folder beside the executable, else the source tree's studio/frontend/dist. The
// native folder is --native-exe's folder, else "native" beside the executable, else the
// source tree's studio/native/build/Release.
func resolvePaths(options launcher.Options) paths {
	executable, _ := os.Executable()
	folder := filepath.Dir(executable)
	result := paths{data: options.DataDir}
	if result.data == "" {
		result.data = filepath.Join(folder, "data")
		if info, err := os.Stat(result.data); (err != nil || !info.IsDir()) && os.Getenv("LOCALAPPDATA") != "" {
			result.data = filepath.Join(os.Getenv("LOCALAPPDATA"), "ScadaBench", "data")
		}
	}
	result.frontend = first(append([]string{filepath.Join(folder, "frontend.pak"), filepath.Join(folder, "frontend")},
		upward(folder, filepath.Join("frontend", "dist"))...))
	if options.NativeExe != "" {
		result.native = filepath.Dir(options.NativeExe)
	} else if found := first(append([]string{filepath.Join(folder, "native", "studio_ied.exe")},
		upward(folder, filepath.Join("native", "build", "Release", "studio_ied.exe"))...)); found != "" {
		result.native = filepath.Dir(found)
	} else {
		result.native = filepath.Join(folder, "native")
	}
	// A development binary built outside the tree (go build -o <temp>) finds the bundled
	// XSDs from the working directory.
	working, _ := os.Getwd()
	if found := first(append(append([]string{filepath.Join(folder, "schemas", "SCL2007B4")},
		upward(folder, filepath.Join("schemas", "SCL2007B4"))...),
		upward(working, filepath.Join("studio", "schemas", "SCL2007B4"))...)); found != "" {
		result.schemas = filepath.Dir(found)
	}
	result.version = version
	if result.version == "" {
		result.version = sourceVersion(folder)
	}
	return result
}

// upward lists folder/relative for folder and each of its parents.
func upward(folder, relative string) []string {
	var candidates []string
	for dir := folder; ; dir = filepath.Dir(dir) {
		candidates = append(candidates, filepath.Join(dir, relative))
		if filepath.Dir(dir) == dir {
			return candidates
		}
	}
}

// first returns the first existing path, or "".
func first(candidates []string) string {
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

var versionLine = regexp.MustCompile(`(?m)^__version__\s*=\s*"([^"]+)"`)

// sourceVersion reads __version__ from backend/studio_app/version.py above folder.
func sourceVersion(folder string) string {
	file := first(upward(folder, filepath.Join("backend", "studio_app", "version.py")))
	if file == "" {
		return "0.0.0+dev"
	}
	data, err := os.ReadFile(file)
	if match := versionLine.FindSubmatch(data); err == nil && match != nil {
		return string(match[1])
	}
	return "0.0.0+dev"
}

// frontendReady is entrypoint.frontend_ready: a TCP connection to the Vite address within
// timeout, retried every 0.25 s.
func frontendReady(address string, timeout time.Duration) bool {
	parsed, err := url.Parse(address)
	if err != nil {
		return false
	}
	port := parsed.Port()
	if port == "" {
		port = "80"
		if parsed.Scheme == "https" {
			port = "443"
		}
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", net.JoinHostPort(parsed.Hostname(), port), time.Second)
		if err == nil {
			_ = connection.Close()
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}
