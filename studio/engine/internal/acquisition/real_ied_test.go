package acquisition

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

const analog = "ReportTestCTRL/GGIO1.AnIn1.mag.f"

// standIn runs testdata/rig/stand_in.py: a studio_ied device on 127.64.0.31:20131 and an
// independent studio_live client driven by JSON lines.
type standIn struct {
	cmd   *exec.Cmd
	in    *json.Encoder
	out   *bufio.Scanner
	model *pyjson.Object
}

func startStandIn(t *testing.T, root string) *standIn {
	python := os.Getenv("STUDIO_PYTHON")
	modelFile := filepath.Join(t.TempDir(), "model.json")
	cmd := exec.Command(python, filepath.Join("testdata", "rig", "stand_in.py"), modelFile)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	rig := &standIn{cmd: cmd, in: json.NewEncoder(stdin), out: bufio.NewScanner(stdout)}
	rig.out.Buffer(make([]byte, 1<<20), 16<<20)
	if !rig.out.Scan() {
		t.Fatal("stand-in did not start")
	}
	data, err := os.ReadFile(modelFile)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _ := pyjson.Decode(data)
	rig.model = pyv.Obj(decoded)
	t.Cleanup(func() { _ = stdin.Close(); _ = cmd.Wait() })
	return rig
}

func (r *standIn) call(t *testing.T, command map[string]any) *pyjson.Object {
	if err := r.in.Encode(command); err != nil || !r.out.Scan() {
		t.Fatalf("stand-in %v: %v", command, err)
	}
	decoded, _ := pyjson.Decode(r.out.Bytes())
	answer := pyv.Obj(decoded)
	if !pyv.Truthy(pyv.Get(answer, "ok")) {
		t.Fatalf("stand-in %v: %s", command, r.out.Text())
	}
	return pyv.GetObj(answer, "result")
}

// TestRealIEDSession connects a Go session through studio_live to a separate simulated
// device and checks values, report use and release with an independent client.
// Run with STUDIO_REAL_IED=1 and STUDIO_PYTHON=<studio venv python>.
func TestRealIEDSession(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", "..", "..", "native", "build", "Release"))
	live := filepath.Join(root, "studio_live.exe")
	if os.Getenv("STUDIO_REAL_IED") != "1" {
		t.Skip("set STUDIO_REAL_IED=1 and STUDIO_PYTHON to run against studio_ied/studio_live")
	}
	rig := startStandIn(t, root)
	ied := pyv.Shallow(rig.model)
	ied.Set("id", "real1").Set("role", "real").Set("live", pyv.O("ip", "127.64.0.31", "port", int64(20131),
		"access", "auto_reports", "poll_ms", int64(200)))
	project := pyv.O("ieds", []any{ied}, "diagram", pyv.O("nodes", []any{}),
		"test_panel", []any{pyv.O("ied_id", "real1", "ref", analog)}, "network", pyv.O("mode", "loopback"))
	values, events := newTestValues(project), &testEvents{}
	deps := Deps{LiveExecutable: live, Start: startPipe, Projects: testProjects{project}, Runtime: testRuntime{"running"},
		Events: events, Bus: events, Values: values, Samples: testIdle{}, Equipment: testIdle{}, States: testIdle{},
		Network: testIdle{}, Clock: testClock{start: time.Now()}}
	manager := NewManager(deps)
	began := time.Now()
	if err := manager.Start(project, nil); err != nil {
		t.Fatal(err)
	}
	defer manager.Stop()
	session, _ := manager.Session("real1")
	waitFor(t, 60*time.Second, func() bool { return pyv.GetStr(session.Status(), "state") == "live" })
	t.Logf("connected, discovered and live after %v", time.Since(began))
	status := session.Status()
	summary := pyv.GetObj(pyv.GetObj(status, "comparison"), "summary")
	if !pyv.Equal(pyv.Get(summary, "missing_on_device"), 0) || !pyv.Equal(pyv.Get(summary, "type_mismatch"), 0) {
		t.Fatalf("comparison: %v", pyv.Repr(summary))
	}
	t.Logf("status: %s", pyv.Repr(pyv.Merge(status, pyv.O("comparison", summary))))
	rig.call(t, map[string]any{"op": "update", "ref": analog, "value": 42.5})
	waitFor(t, 10*time.Second, func() bool { value, _, _ := values.Get("real1", analog); return pyv.Equal(value, 42.5) })
	independent := rig.call(t, map[string]any{"op": "read", "refs": []any{map[string]any{"ref": analog, "fc": "MX"}}})
	if got := pyv.Get(pyv.Objects(pyv.GetList(independent, "values"))[0], "value"); !pyv.Equal(got, 42.5) {
		t.Fatalf("independent client read %v", got)
	}
	rcbs := []any{"ReportTestCTRL/LLN0.RP.EventsRCB"}
	held := pyv.Objects(pyv.GetList(rig.call(t, map[string]any{"op": "rcb_status", "rcbs": rcbs}), "blocks"))[0]
	t.Logf("while Go holds it, the independent client sees: %s", pyv.Repr(held))
	health := manager.StationHealth()
	t.Logf("station verdict %s; refresh failures %s", pyv.Repr(pyv.Get(health, "verdict")), pyv.Repr(manager.RefreshHealth()))
	manager.Stop()
	released := pyv.Objects(pyv.GetList(rig.call(t, map[string]any{"op": "rcb_status", "rcbs": rcbs}), "blocks"))[0]
	if !pyv.Truthy(pyv.Get(released, "free")) {
		t.Fatalf("report block not given back: %s", pyv.Repr(released))
	}
	t.Logf("after Stop the independent client sees: %s", pyv.Repr(released))
	t.Logf("%d events, first: %v", len(events.all()), events.all()[:3])
}

func waitFor(t *testing.T, limit time.Duration, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(limit)
	for !done() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
