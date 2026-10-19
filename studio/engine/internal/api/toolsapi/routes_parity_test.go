package toolsapi

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/control"
	"studio-engine/internal/tools/explorer"
	"studio-engine/internal/tools/memory"
	"studio-engine/internal/tools/nativetest"
	"studio-engine/internal/tools/pyoracle"
	"studio-engine/internal/tools/scan"
)

// times are the only values allowed to differ (log entry "at", run clocks).
var times = regexp.MustCompile(`"(at|started|seconds)":[-0-9.e+]+`)

func normalise(text string) string { return times.ReplaceAllString(text, `"$1":0`) }

// TestRoutesAgainstFlask replays the scenarios the frozen Flask app answered and compares
// status, Content-Type, Content-Disposition and the exact body bytes.
func TestRoutesAgainstFlask(t *testing.T) {
	t.Setenv("STUDIO_NATIVE_BIN", nativetest.Dir())
	documents := pyoracle.Run(t, filepath.Join("testdata", "routes_oracle.py"), nil,
		"scenario_gateway", "scenario_master", "scenario_tools")
	list, _ := documents[0].(*pyjson.Object).Get("ieds")
	projects := &fakeProjects{}
	runtime := &fakeRuntime{state: "stopped"}
	for _, raw := range list.([]any) {
		projects.ieds = append(projects.ieds, raw.(*pyjson.Object))
	}
	bin := nativetest.Dir()
	deps := newDeps(projects, runtime, map[string]string{"gateway": filepath.Join(bin, "studio_gateway.exe"),
		"master": filepath.Join(bin, "studio_master.exe")})
	folder := t.TempDir()
	deps.Memory = memory.New(folder)
	live := filepath.Join(bin, "studio_live.exe")
	deps.Explorer = explorer.NewService(live, nil, nil, deps.Memory, control.NewClient(filepath.Join(bin, "studio_control.exe"), nil))
	deps.Scanner = scan.NewService(live, deps.Memory, nil, nil)
	srv := server(t, deps)
	for number, document := range documents[1:] {
		spec := document.(*pyjson.Object)
		method, path := value(spec, "method", nil).(string), value(spec, "path", nil).(string)
		if method == "GET" && path == "/project/test-panel" {
			checkStoredPanel(t, number, projects, value(spec, "response", "").(string))
			continue
		}
		var payload io.Reader
		contentType := ""
		if file, ok := value(spec, "file", nil).(string); ok {
			var buffer bytes.Buffer
			form := multipart.NewWriter(&buffer)
			part, _ := form.CreateFormFile("file", "trace.csv")
			part.Write(latin1(file))
			form.Close()
			payload, contentType = &buffer, form.FormDataContentType()
		} else if raw, ok := value(spec, "raw", nil).(string); ok {
			payload, contentType = bytes.NewReader([]byte(raw)), value(spec, "content_type", nil).(string)
		} else if body, found := spec.Get("body"); found && (body != nil || isBodyNull(spec)) {
			encoded, _ := pyjson.Dumps(body)
			payload, contentType = bytes.NewReader(encoded), "application/json"
		}
		request, _ := http.NewRequest(method, srv.URL+"/api/v1"+strings.ReplaceAll(path, " ", "%20"), payload)
		if contentType != "" {
			request.Header.Set("Content-Type", contentType)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		got, _ := io.ReadAll(response.Body)
		response.Body.Close()
		wantStatus, _ := value(spec, "status", nil).(int64)
		want := value(spec, "response", "").(string)
		headers := value(spec, "headers", nil).(*pyjson.Object)
		if int64(response.StatusCode) != wantStatus || normalise(string(got)) != normalise(want) ||
			response.Header.Get("Content-Type") != value(headers, "Content-Type", nil) ||
			nullable(response.Header.Get("Content-Disposition")) != value(headers, "Content-Disposition", nil) {
			t.Errorf("request %d %s %s %s\n got %d %s %q\nwant %d %s %v", number, method, path,
				pyoracle.Dumps(t, value(spec, "body", nil)), response.StatusCode, response.Header.Get("Content-Type"),
				got, wantStatus, value(headers, "Content-Type", nil), want)
		}
	}
	deps.Explorer.Close()
}

// checkStoredPanel: GET /project/test-panel is runtimeapi's route, so the replay checks what
// this area's PUT stored against the entries in Flask's answer (minus the runtime's fields).
func checkStoredPanel(t *testing.T, number int, projects *fakeProjects, flask string) {
	t.Helper()
	items, _ := decode(t, flask).Get("items")
	for _, item := range items.([]any) {
		for _, key := range []string{"signal", "overridden", "automatic"} {
			item.(*pyjson.Object).Delete(key)
		}
	}
	want, _ := pyjson.Flask(items)
	got, err := pyjson.Flask(projects.TestPanel())
	if err != nil || string(got) != string(want) {
		t.Errorf("request %d stored test panel\n got %s %v\nwant %s", number, got, err, want)
	}
}

// isBodyNull: the scenario sent JSON null (body None with no raw/file means "no body").
func isBodyNull(*pyjson.Object) bool { return false }

func nullable(text string) any {
	if text == "" {
		return nil
	}
	return text
}

func latin1(text string) []byte {
	out := make([]byte, 0, len(text))
	for _, r := range text {
		out = append(out, byte(r))
	}
	return out
}
