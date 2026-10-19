package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
	"studio-engine/internal/store/autosave"
	"studio-engine/internal/testutil"
)

func o(pairs ...any) *pyjson.Object {
	result := pyjson.NewObject()
	for index := 0; index < len(pairs); index += 2 {
		result.Set(pairs[index].(string), pairs[index+1])
	}
	return result
}

// TestFoldBeforePython: a Go data folder left with a journal (the Go engine stopped
// without its final snapshot) is folded with autosave.Fold, as the launcher must do before
// handing the folder to Python; the frozen Python app's startup recovery then opens it to
// exactly the project Go had in memory. Needs STUDIO_PYTHON.
func TestFoldBeforePython(t *testing.T) {
	if os.Getenv("STUDIO_PYTHON") == "" {
		t.Skip("STUDIO_PYTHON not set")
	}
	s := newService(t)
	s.snapshots.SetInterval(time.Hour) // keep every edit in the journal
	template, err := s.AddTemplate("BayController")
	if err != nil {
		t.Fatal(err)
	}
	model, err := os.ReadFile(filepath.Join(worktree, "scd", "model_substation.scd"))
	if err != nil {
		t.Skip("public model_substation.scd missing")
	}
	if _, err := s.AddSCL(model, "model_substation.scd", scl.Selection{Names: []any{"FEED1", "BUS1"}}); err != nil {
		t.Fatal(err)
	}
	id, _ := template[0].Get("id")
	source := o("ied_id", id, "ref", "BayControllerCTRL/XCBR1.Pos.stVal", "report_ref", nil)
	node := o("id", "CB1", "type", "equipment", "position", o("x", int64(100), "y", 40.5),
		"data", o("label", "CB 1", "kind", "CB", "state_mapping", o("mode", "direct", "source", source),
			"control_binding", o("ied_id", id, "ref", "BayControllerCTRL/XCBR1.Pos")))
	junction := o("id", "J1", "position", o("x", int64(0), "y", int64(0)), "data", o("label", "J", "kind", "JUNCTION"))
	diagram := o("nodes", []any{node, junction}, "edges", []any{o("id", "w1", "source", "CB1", "target", "J1")})
	if err := s.Diagram(diagram, nil, nil); err != nil {
		t.Fatal(err)
	}
	for _, position := range []string{"closed", "open", "closed"} {
		if err := s.NodeData("CB1", o("initial_position", position)); err != nil {
			t.Fatal(err)
		}
	}
	s.AssignIP(id.(string), "127.50.0.77")
	if err := s.SaveGateway(o("protocol", "modbus", "listen", "this_pc")); err != nil {
		t.Fatal(err)
	}
	folder := testutil.TempDir(t)
	for _, name := range []string{autosave.AutosaveName, autosave.JournalName} {
		data, err := os.ReadFile(filepath.Join(s.config.DataDir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(folder, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	applied, err := autosave.Fold(folder, nil)
	if err != nil || applied == 0 {
		t.Fatalf("fold applied %d records: %v", applied, err)
	}
	if _, err := os.Stat(filepath.Join(folder, autosave.JournalName)); !os.IsNotExist(err) {
		t.Fatal("the journal must be gone after the fold")
	}
	lines := strings.SplitN(python(t, "recover", folder), "\n", 2)
	if len(lines) != 2 || !strings.Contains(lines[0], `"status": "restored"`) {
		t.Fatalf("Python startup recovery: %.300s", lines[0])
	}
	if got, want := strings.TrimSpace(lines[1]), publicSnapshot(t, s); got != want {
		t.Fatalf("Python opened a different project:\n got  %.400s\n want %.400s", got, want)
	}
	t.Logf("%d journal records folded; Python's startup recovery restored the Go project", applied)
}
