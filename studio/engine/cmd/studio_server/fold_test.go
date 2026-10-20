package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/testutil"
)

// api sends one request with the dev token and returns the decoded JSON answer.
func api(t *testing.T, url, method, path, body string) *pyjson.Object {
	t.Helper()
	request, _ := http.NewRequest(method, url+path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer tok")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)
	if response.StatusCode != 200 {
		t.Fatalf("%s %s: %d %s", method, path, response.StatusCode, data)
	}
	decoded, err := pyjson.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return decoded.(*pyjson.Object)
}

// canonical drops the per-process and runtime-only keys and sorts the rest (jsonify order).
func canonical(t *testing.T, snapshot *pyjson.Object) string {
	for _, key := range []string{"session_id", "revision", "runtime", "running_ieds", "startup_recovery"} {
		snapshot.Delete(key)
	}
	data, err := pyjson.Flask(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes.TrimSpace(data))
}

// TestFoldBeforePython: the composed server is killed hard after edits (its journal stays
// behind), "studio_server --fold-journal" folds it as run_dev_engine.py does before
// starting Python, and the frozen Python startup recovery opens exactly what Go served.
func TestFoldBeforePython(t *testing.T) {
	python := os.Getenv("STUDIO_PYTHON")
	if python == "" || testing.Short() {
		t.Skip("needs STUDIO_PYTHON and builds the server")
	}
	binary, data := build(t), testutil.TempDir(t)
	seed, _ := filepath.Abs(filepath.Join("..", "..", "internal", "scl", "testdata", "golden", "xsd_seed.json"))
	t.Setenv("STUDIO_XSD_SEED", seed)
	t.Setenv("SCADABENCH_UPDATE_CHECK", "off")
	t.Setenv("LOCALAPPDATA", testutil.TempDir(t)) // license home of this test only
	port := highPort(t)
	server, _ := start(t, binary, 1, "--dev", "--dev-token", "tok", "--data-dir", data, "--port", strconv.Itoa(port),
		"--native-exe", filepath.Join(testutil.TempDir(t), "studio_ied.exe"), "--frontend-url", "http://127.0.0.1:1")
	url := "http://127.0.0.1:" + strconv.Itoa(port)
	ids, _ := api(t, url, "POST", "/api/v1/ieds/template", `{"name": "BayController"}`).Get("ieds")
	id := ids.([]any)[0].(string)
	revision, _ := api(t, url, "GET", "/api/v1/project", "").Get("revision")
	diagram := `{"nodes": [{"id": "CB1", "type": "equipment", "position": {"x": 100, "y": 40.5}, "data": {"label": "CB 1",
		"kind": "CB", "state_mapping": {"mode": "direct", "source": {"ied_id": "` + id + `", "ref":
		"BayControllerCTRL/XCBR1.Pos.stVal", "report_ref": null}}, "control_binding": {"ied_id": "` + id + `", "ref":
		"BayControllerCTRL/XCBR1.Pos"}}}], "edges": []}`
	api(t, url, "PUT", "/api/v1/project/diagram", `{"diagram": `+diagram+`, "revision": `+pyjsonText(revision)+`, "tab_id": "t1"}`)
	api(t, url, "PUT", "/api/v1/ieds/"+id+"/settings", `{"port": 20777}`)
	want := canonical(t, api(t, url, "GET", "/api/v1/project", ""))
	_ = server.Process.Kill() // no final snapshot: the journal holds the edits
	_ = server.Wait()
	if _, err := os.Stat(filepath.Join(data, "autosave.journal")); err != nil {
		t.Fatalf("a hard-killed Go server must leave its journal: %v", err)
	}
	if output, err := exec.Command(binary, "--fold-journal", data).CombinedOutput(); err != nil {
		t.Fatalf("--fold-journal: %v %s", err, output)
	}
	if _, err := os.Stat(filepath.Join(data, "autosave.journal")); !os.IsNotExist(err) {
		t.Fatal("the journal must be gone after --fold-journal")
	}
	script := filepath.Join("..", "..", "internal", "project", "service", "testdata", "roundtrip.py")
	output, err := exec.Command(python, "-I", script, "recover", data).Output()
	if err != nil {
		t.Fatalf("python recover: %v %s", err, output)
	}
	lines := strings.SplitN(strings.TrimSpace(string(output)), "\n", 2)
	if len(lines) != 2 || !strings.Contains(lines[0], `"status": "restored"`) {
		t.Fatalf("Python startup recovery: %.300s", output)
	}
	restored, err := pyjson.Decode([]byte(lines[1]))
	if err != nil {
		t.Fatal(err)
	}
	if got := canonical(t, restored.(*pyjson.Object)); got != want {
		t.Fatalf("Python opened a different project:\n got  %.600s\n want %.600s", got, want)
	}
}

func pyjsonText(value any) string {
	data, _ := pyjson.Dumps(value)
	return string(data)
}
