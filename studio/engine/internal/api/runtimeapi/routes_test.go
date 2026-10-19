package runtimeapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime"
)

func newServer(t *testing.T, portBase int64) (*httptest.Server, *runtime.Coordinator, *runtime.MemoryProject) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "runtime", "testdata", "project.json"))
	if err != nil {
		t.Fatal(err)
	}
	decoded, _ := pyjson.Decode(raw)
	project := decoded.(*pyjson.Object)
	ieds, _ := project.Get("ieds")
	for i, item := range ieds.([]any) {
		item.(*pyjson.Object).Set("port", portBase+int64(i)+1)
	}
	native, _ := filepath.Abs(filepath.Join("..", "..", "..", "..", "native", "build", "Release"))
	memory := runtime.NewMemoryProject(project)
	coordinator := runtime.New(runtime.Options{NativeDir: native, DataDir: t.TempDir(), Bus: &runtime.MemoryBus{}, Project: memory})
	t.Cleanup(coordinator.Close)
	mux := http.NewServeMux()
	for _, route := range (Area{Runtime: coordinator, Project: memory}).Routes() {
		mux.Handle(route.Method+" "+route.Pattern, route.Handler)
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, coordinator, memory
}

// call sends one request and returns status and body text.
func call(t *testing.T, server *httptest.Server, method, path, contentType, body string) (int, string) {
	t.Helper()
	request, _ := http.NewRequest(method, server.URL+"/api/v1"+path, strings.NewReader(body))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	text, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(text)
}

// TestStoppedAnswersMatchFlask replays testdata/stopped_requests.json and compares status and
// exact body bytes with the Python engine's answers (testdata/record_flask.py).
func TestStoppedAnswersMatchFlask(t *testing.T) {
	server, _, _ := newServer(t, 21300)
	var requests [][4]string
	var golden []struct {
		Status int
		Body   string
	}
	rawRequests, _ := os.ReadFile(filepath.Join("testdata", "stopped_requests.json"))
	rawGolden, _ := os.ReadFile(filepath.Join("testdata", "stopped_golden.json"))
	if json.Unmarshal(rawRequests, &requests) != nil {
		t.Fatal("bad stopped_requests.json")
	}
	var pairs [][2]any
	if json.Unmarshal(rawGolden, &pairs) != nil || len(pairs) != len(requests) {
		t.Fatal("stopped_golden.json does not match the requests; run record_flask.py")
	}
	for _, pair := range pairs {
		golden = append(golden, struct {
			Status int
			Body   string
		}{int(pair[0].(float64)), pair[1].(string)})
	}
	for i, request := range requests {
		status, body := call(t, server, request[0], request[1], request[2], request[3])
		if status != golden[i].Status || body != golden[i].Body {
			t.Errorf("%s %s %s: %d %q, Python %d %q", request[0], request[1], request[3], status, body,
				golden[i].Status, golden[i].Body)
		}
	}
}

func TestRunOperateAndStopOverHTTP(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "..", "..", "..", "native", "build", "Release", "studio_ied.exe")); err != nil {
		t.Skip("native build required")
	}
	server, coordinator, _ := newServer(t, 21310)
	status, body := call(t, server, "POST", "/runtime/start", "", "")
	if status != 200 || !strings.Contains(body, `"state":"running"`) {
		t.Fatalf("start: %d %s", status, body)
	}
	status, body = call(t, server, "POST", "/equipment/cb1/operate", "application/json", `{"action":"close"}`)
	if status != 200 {
		t.Fatalf("operate: %d %s", status, body)
	}
	var result map[string]any
	_ = json.Unmarshal([]byte(body), &result)
	if result["action"] != "close" || result["wire_value"] != true {
		t.Fatalf("operate result %s", body)
	}
	if command, _ := result["command"].(map[string]any); command["ok"] != true {
		t.Fatalf("command %s", body)
	}
	status, body = call(t, server, "GET", "/equipment/states", "", "")
	if status != 200 || !strings.Contains(body, `{"node_id":"cb1","value":2}`) {
		t.Fatalf("states: %d %s", status, body)
	}
	status, body = call(t, server, "POST", "/runtime/stop", "", "")
	if status != 200 || body != "{\"running\":[],\"state\":\"stopped\"}\n" {
		t.Fatalf("stop: %d %s", status, body)
	}
	if coordinator.State() != "stopped" {
		t.Fatal("not stopped")
	}
}
