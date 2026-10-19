package main

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func build(t *testing.T) string {
	binary := filepath.Join(t.TempDir(), "studio_server.exe")
	command := exec.Command("go", "build", "-o", binary, ".")
	command.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, output)
	}
	return binary
}

func highPort(t *testing.T) int {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	if port < 20000 {
		port = 20000 + port%20000
	}
	return port
}

// start runs the server and returns its first output lines.
func start(t *testing.T, binary string, lines int, args ...string) (*exec.Cmd, []string) {
	command := exec.Command(binary, args...)
	stdout, _ := command.StdoutPipe()
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = command.Process.Kill(); _ = command.Wait() })
	reader := bufio.NewReader(stdout)
	var out []string
	for len(out) < lines {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("server output ended after %q: %v", out, err)
		}
		out = append(out, strings.TrimRight(line, "\r\n"))
	}
	return command, out
}

// TestServerModeLikePythonLauncher starts the built server as run_dev_engine.py does and
// checks the printed lines, launcher.json, the session through the Vite origin, SSE and
// the data-folder lease.
func TestServerModeLikePythonLauncher(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the server")
	}
	binary, data := build(t), t.TempDir()
	port, vite := highPort(t), "http://127.0.0.1:"+strconv.Itoa(highPort(t))
	_, lines := start(t, binary, 1, "--dev", "--dev-token", "tok", "--data-dir", data,
		"--port", strconv.Itoa(port), "--frontend-url", vite)
	url := "http://127.0.0.1:" + strconv.Itoa(port)
	if lines[0] != "ScadaBench: "+url {
		t.Fatalf("readiness line %q", lines[0])
	}
	if record, _ := os.ReadFile(filepath.Join(data, "launcher.json")); string(record) != `{"url": "`+url+`"}` {
		t.Errorf("launcher.json %q", record)
	}
	request, _ := http.NewRequest("POST", url+"/api/v1/session", strings.NewReader(`{"token": "tok"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", vite)
	response, err := http.DefaultClient.Do(request)
	if err != nil || response.StatusCode != 200 {
		t.Fatalf("dev session through the Vite origin: %v %v", response, err)
	}
	cookie := response.Header.Get("Set-Cookie")
	response.Body.Close()
	events, _ := http.NewRequest("GET", url+"/api/v1/events", nil)
	events.Header.Set("Cookie", strings.SplitN(cookie, ";", 2)[0])
	stream, err := http.DefaultClient.Do(events)
	if err != nil {
		t.Fatal(err)
	}
	first := make([]byte, len("retry: 1000\n\n"))
	_, _ = io.ReadFull(stream.Body, first)
	stream.Body.Close()
	if string(first) != "retry: 1000\n\n" {
		t.Errorf("event stream starts %q", first)
	}
	second := exec.Command(binary, "--data-dir", data, "--port", strconv.Itoa(highPort(t)))
	output, err := second.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "This ScadaBench data folder is already open") {
		t.Errorf("second launch on the same data folder: %v %q", err, output)
	}
}

func TestPackagedModeLines(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the server")
	}
	binary := build(t)
	busy, _ := net.Listen("tcp4", "127.0.0.1:0")
	defer busy.Close()
	_, lines := start(t, binary, 2, "--no-browser", "--data-dir", t.TempDir(),
		"--port", strconv.Itoa(busy.Addr().(*net.TCPAddr).Port))
	if !regexp.MustCompile(`^ScadaBench: http://127\.0\.0\.1:\d+$`).MatchString(lines[0]) ||
		strings.HasSuffix(lines[0], ":"+strconv.Itoa(busy.Addr().(*net.TCPAddr).Port)) {
		t.Errorf("busy port must fall back to a free one: %q", lines[0])
	}
	url := strings.TrimPrefix(lines[0], "ScadaBench: ")
	if !regexp.MustCompile(`^Open in your browser: ` + regexp.QuoteMeta(url) + `#session=[A-Za-z0-9_-]{43}$`).MatchString(lines[1]) {
		t.Errorf("open line %q", lines[1])
	}
}
