package main

import (
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"studio-engine/internal/launcher"
)

type paths struct {
	data     string
	frontend string
}

// resolvePaths mirrors Settings.defaults: --data-dir wins; otherwise a "data" folder beside
// the executable, else %LOCALAPPDATA%\ScadaBench\data. The frontend is frontend.pak or the
// frontend folder beside the executable, else the source tree's studio/frontend/dist.
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
	candidates := []string{filepath.Join(folder, "frontend.pak"), filepath.Join(folder, "frontend")}
	for dir := folder; ; dir = filepath.Dir(dir) {
		candidates = append(candidates, filepath.Join(dir, "frontend", "dist"))
		if filepath.Dir(dir) == dir {
			break
		}
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			result.frontend = candidate
			break
		}
	}
	return result
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
