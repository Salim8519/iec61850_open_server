package app

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// variant is one way the Python app was created (dev or packaged security, license and
// startup state); the Go app is composed with the same values.
type variant struct {
	name    string
	origin  string // AdminOrigin (dev); empty = packaged rules
	license string // "none" or "unlicensed"
	startup string // "ok", "timeout" or "error"
}

const parityToken, paritySecret = "parity-token", "parity-secret"

type fakeLicense struct{ enforce bool }

func (f fakeLicense) Enforced() bool { return f.enforce }
func (f fakeLicense) Licensed() bool { return false }

type fakeStartup struct {
	ready bool
	err   error
}

func (f fakeStartup) Wait(time.Duration) bool { return f.ready }
func (f fakeStartup) Err() error              { return f.err }

// startPython runs testdata/flask_server.py and returns its base URL; the process is
// killed when the test ends.
func startPython(t *testing.T, v variant, frontend string) string {
	t.Helper()
	python := os.Getenv("STUDIO_PYTHON")
	if python == "" {
		t.Skip("set STUDIO_PYTHON to a Python with the studio requirements to run the Flask parity test")
	}
	backend, _ := filepath.Abs(filepath.Join("..", "..", "..", "backend"))
	args := []string{"-I", filepath.Join("testdata", "flask_server.py"), "--backend", backend,
		"--port", strconv.Itoa(freePort(t)), "--data-dir", t.TempDir(), "--token", parityToken,
		"--secret", paritySecret, "--frontend", frontend, "--license", v.license, "--startup", v.startup}
	if v.origin != "" {
		args = append(args, "--admin-origin", v.origin)
	}
	command := exec.Command(python, args...)
	var logs bytes.Buffer // Flask logs every 500 with a traceback; shown only on failure
	command.Stderr = &logs
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("Python stderr:\n%s", logs.String())
		}
	})
	output, err := command.StdoutPipe()
	if err != nil || command.Start() != nil {
		t.Fatalf("start python: %v", err)
	}
	t.Cleanup(func() { _ = command.Process.Kill(); _ = command.Wait() })
	ready := make(chan string, 1)
	go func() {
		lines := bufio.NewScanner(output)
		for lines.Scan() {
			if url, found := strings.CutPrefix(lines.Text(), "ScadaBench: "); found {
				ready <- url
			}
		}
	}()
	select {
	case url := <-ready:
		return url
	case <-time.After(90 * time.Second):
		t.Fatal("the Python app did not print its readiness line")
	}
	return ""
}

// startGo serves the composed Go app on a loopback port of 20000 or above.
func startGo(t *testing.T, v variant, frontend string) string {
	t.Helper()
	var startupErr error
	if v.startup == "error" {
		startupErr = errors.New("the autosave is a link")
	}
	engine, err := New(Config{DataDir: t.TempDir(), FrontendDir: frontend, Token: parityToken,
		SecretKey: paritySecret, AdminOrigin: v.origin, License: fakeLicense{v.license == "unlicensed"},
		Startup: fakeStartup{ready: v.startup != "timeout", err: startupErr}})
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:"+strconv.Itoa(freePort(t)))
	if err != nil {
		t.Fatal(err)
	}
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: engine.Handler}}
	server.Start()
	t.Cleanup(server.Close)
	return server.URL
}

func freePort(t *testing.T) int {
	for attempt := 0; attempt < 50; attempt++ {
		port := 20000 + rand.Intn(20000)
		if listener, err := net.Listen("tcp4", "127.0.0.1:"+strconv.Itoa(port)); err == nil {
			_ = listener.Close()
			return port
		}
	}
	t.Fatal("no free loopback port above 20000")
	return 0
}

// syntheticFrontend writes a small dist folder (index.html and assets) both apps serve.
func syntheticFrontend(t *testing.T) string {
	folder := t.TempDir()
	files := map[string]string{"index.html": "<!doctype html><title>ScadaBench</title>",
		"assets/app-1a2b.js": "console.log('scadabench');\n", "assets/style-3c4d.css": "body{margin:0}\n",
		"assets/logo.svg": "<svg xmlns='http://www.w3.org/2000/svg'/>", "assets/data.bin": "\x00\x01"}
	for name, content := range files {
		path := filepath.Join(folder, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return folder
}

// comparedHeaders are the response headers that must be identical.
var comparedHeaders = []string{"Content-Type", "Content-Length", "Vary", "Allow", "Cache-Control",
	"Accept-Ranges", "ETag", "Last-Modified", "Content-Disposition", "Location"}

// describe renders status, compared headers and body for a diff.
func describe(response *http.Response, body []byte) string {
	var out strings.Builder
	fmt.Fprintf(&out, "%d\n", response.StatusCode)
	for _, name := range comparedHeaders {
		values := response.Header.Values(name)
		if name == "Allow" && len(values) == 1 {
			// Werkzeug lists the methods of a Python set, so their order changes from one
			// Python process to the next; only the set is part of the behaviour.
			methods := strings.Split(values[0], ", ")
			sort.Strings(methods)
			values = []string{strings.Join(methods, ", ")}
		}
		if len(values) > 0 {
			fmt.Fprintf(&out, "%s: %s\n", name, strings.Join(values, " | "))
		}
	}
	if cookies := response.Header.Values("Set-Cookie"); len(cookies) > 0 {
		fmt.Fprintf(&out, "Set-Cookie: %s\n", normalizeCookie(cookies[0]))
	}
	fmt.Fprintf(&out, "%q", body)
	return out.String()
}

// normalizeCookie hides the timestamp and signature of a session cookie.
func normalizeCookie(header string) string {
	value, attributes, _ := strings.Cut(header, ";")
	parts := strings.Split(value, ".")
	if len(parts) >= 3 {
		parts[len(parts)-2], parts[len(parts)-1] = "<time>", "<signature>"
	}
	return strings.Join(parts, ".") + ";" + attributes
}
