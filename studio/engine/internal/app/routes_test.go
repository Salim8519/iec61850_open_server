package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/api/hostkeyapi"
	"studio-engine/internal/api/labapi"
	"studio-engine/internal/api/liveapi"
	"studio-engine/internal/api/platformapi"
	"studio-engine/internal/api/projectapi"
	"studio-engine/internal/api/runtimeapi"
	"studio-engine/internal/api/snmpapi"
	"studio-engine/internal/api/toolsapi"
	"studio-engine/internal/testutil"
	"studio-engine/internal/tools/explorer"
)

// composeForTest builds the engine on temp folders with no native programs and no
// background loops.
func composeForTest(t *testing.T) *Engine {
	t.Helper()
	schemas, _ := filepath.Abs(filepath.Join("..", "..", "..", "schemas"))
	engine, err := Compose(Options{DataDir: testutil.TempDir(t), NativeDir: testutil.TempDir(t),
		LicenseHome: testutil.TempDir(t), SchemasDir: schemas, Token: "token", SecretKey: "secret",
		Version: "0.0.0+dev", NoBackground: true, Logf: t.Logf})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	return engine
}

var werkzeugVariable = regexp.MustCompile(`<(?:(\w+):)?(\w+)>`)

// goPattern turns a Werkzeug rule into the Go route pattern (<path:x> matches the rest).
func goPattern(rule string) string {
	return werkzeugVariable.ReplaceAllStringFunc(rule, func(match string) string {
		parts := werkzeugVariable.FindStringSubmatch(match)
		if parts[1] == "path" {
			return "{" + parts[2] + "...}"
		}
		return "{" + parts[2] + "}"
	})
}

// flaskRoutes reads the frozen Python app's url_map through STUDIO_PYTHON.
func flaskRoutes(t *testing.T) map[string]bool {
	python := os.Getenv("STUDIO_PYTHON")
	if python == "" {
		t.Skip("set STUDIO_PYTHON to compare the routes with the Flask app")
	}
	backend, _ := filepath.Abs(filepath.Join("..", "..", "..", "backend"))
	output, err := exec.Command(python, "-I", filepath.Join("testdata", "url_map.py"), "--backend", backend,
		"--data-dir", testutil.TempDir(t)).Output()
	if err != nil {
		t.Fatalf("url_map.py: %v", err)
	}
	var rules [][2]string
	if err := json.Unmarshal(output, &rules); err != nil {
		t.Fatalf("url_map.py output: %v\n%s", err, output)
	}
	routes := map[string]bool{}
	for _, rule := range rules {
		routes[rule[0]+" "+goPattern(rule[1])] = true
	}
	return routes
}

// TestEveryPythonRouteServed: the composed engine serves exactly Flask's routes (plus the
// Go-only push and own-point routes), none of them is a 501 placeholder, and every GET route answers
// something other than 501.
func TestEveryPythonRouteServed(t *testing.T) {
	want := flaskRoutes(t)
	engine := composeForTest(t)
	served := map[string]bool{"POST /api/v1/session": true, "GET /": true, "GET /assets/{filename...}": true,
		"GET /api/v1/events": true}
	for _, route := range engine.Routes {
		key := route.Method + " " + route.Pattern
		if served[key] {
			t.Errorf("%s is served twice", key)
		}
		served[key] = true
	}
	var missing, extra []string
	for key := range want {
		if !served[key] {
			missing = append(missing, key)
		}
	}
	for key := range served {
		if !want[key] && !PushOnlyRoutes[key] && !toolsapi.OwnPointRoutes[key] && !toolsapi.MasterGoOnlyRoutes[key] && !snmpapi.GoOnly(key) && !hostkeyapi.GoOnly(key) && !labapi.GoOnly(key) &&
			!platformapi.GoOnlyRoutes[key] && !projectapi.GoOnlyRoutes[key] && !runtimeapi.GoOnlyRoutes[key] &&
			!liveapi.GoOnlyRoutes[key] &&
			key != explorer.CommandRoute {
			extra = append(extra, key)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 || len(extra) > 0 {
		t.Fatalf("routes differ from Flask\nmissing in Go: %v\nonly in Go: %v", missing, extra)
	}
	for key := range want {
		method, pattern, _ := strings.Cut(key, " ")
		if method != "GET" || pattern == "/api/v1/events" { // the event stream never ends
			continue
		}
		path := regexp.MustCompile(`\{(\w+)(\.\.\.)?\}`).ReplaceAllString(pattern, "x")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		request := httptest.NewRequest(method, path, nil).WithContext(ctx)
		request.RemoteAddr, request.Host = "127.0.0.1:50000", "127.0.0.1:8765"
		request.Header.Set("Authorization", "Bearer token")
		recorder := httptest.NewRecorder()
		engine.Handler.ServeHTTP(recorder, request)
		cancel()
		if recorder.Code == http.StatusNotImplemented {
			t.Errorf("%s answered 501: %s", key, recorder.Body.String())
		}
	}
	t.Logf("%d Flask routes, all served by the composed engine", len(want))
}
