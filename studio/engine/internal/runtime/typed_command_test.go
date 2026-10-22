package runtime

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/scl"
	"studio-engine/internal/testutil"
)

type validSchema struct{}

func (validSchema) Validate([]byte, string) (scl.SchemaVerdict, error) {
	return scl.SchemaVerdict{Status: "valid"}, nil
}

// typedRun starts the template project plus the SampleTyped IED (ADR-0104) on real workers.
func typedRun(t *testing.T) (*Coordinator, string) {
	t.Helper()
	folder := nativeDir(t)
	requireMux(t, folder)
	source, err := os.ReadFile(filepath.Join("..", "..", "..", "tests", "fixtures", "typed_controls", "SampleTyped.scd"))
	if err != nil {
		t.Fatal(err)
	}
	importer := &scl.Importer{Validator: validSchema{}, NewID: func() string { return "typed1" }}
	models, err := importer.Import(source, "SampleTyped.scd", scl.SelectAll)
	if err != nil {
		t.Fatal(err)
	}
	ied := models[0].Set("ip", "127.63.1.21").Set("port", int64(21121))
	project := loadProject(t, 21140)
	project.Set("ieds", append(pyval.List(project, "ieds"), ied))
	c := New(Options{NativeDir: folder, DataDir: testutil.TempDir(t), Bus: &MemoryBus{}, Project: NewMemoryProject(project)})
	t.Cleanup(c.Close)
	if err := c.Start([]string{"typed1"}); err != nil {
		t.Fatal(err)
	}
	return c, "typed1"
}

func TestTypedCommandsOnARunningSimulatedIed(t *testing.T) {
	c, id := typedRun(t)
	ref := func(name string) string { return "SampleTypedCTRL/" + name }
	listed, err := c.TypedControls(id)
	if err != nil || len(pyval.List(listed, "extended_controls")) != 24 {
		t.Fatalf("typed list: %v %v", listed, err)
	}
	// ENC by label, direct with enhanced security.
	if _, err := c.OperateTyped(TypedCommand{IEDID: id, Ref: ref("GGIO3.Mod"), Value: "test"}); err != nil {
		t.Fatal(err)
	}
	// BSC higher after an explicit Select (SBO enhanced), originator kept.
	selected, err := c.SelectTyped(TypedCommand{IEDID: id, Ref: ref("ATCC4.TapChg"), Value: "higher",
		Options: CommandOptions{OrIdent: "panel-1", OrCat: 2}})
	if err != nil || selected.Model != "sbo-enhanced" || selected.ExpiresMs != 3000 {
		t.Fatalf("select: %+v %v", selected, err)
	}
	result, err := c.OperateTyped(TypedCommand{IEDID: id, Ref: ref("ATCC4.TapChg"), Value: "higher",
		Options: CommandOptions{OrIdent: "panel-1", OrCat: 2}})
	if err != nil || pyval.Get(result, "terminated") != true {
		t.Fatalf("operate: %v %v", result, err)
	}
	// ISC "higher" moves one position from the present one (5 -> 6).
	if _, err := c.OperateTyped(TypedCommand{IEDID: id, Ref: ref("ATCC1.TapSet"), Value: "higher"}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	for feedback, want := range map[string]any{"GGIO3.Mod.stVal": int64(3), "ATCC4.TapChg.valWTr.posVal": int64(6),
		"ATCC1.TapSet.valWTr.posVal": int64(6)} {
		item := c.runs[id]
		if got, _ := item.table.get(ref(feedback)); !pyval.Equal(got, want) {
			t.Errorf("%s = %v, want %v", feedback, got, want)
		}
	}
	// Select, Cancel, then an Operate with no select works in one step again.
	if _, err := c.SelectTyped(TypedCommand{IEDID: id, Ref: ref("GGIO2.ISCSO1")}); err != nil {
		t.Fatal(err)
	}
	if err := c.CancelTyped(id, ref("GGIO2.ISCSO1")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.OperateTyped(TypedCommand{IEDID: id, Ref: ref("GGIO2.ISCSO1"), Value: int64(4)}); err != nil {
		t.Fatal(err)
	}
}

func TestTypedCommandRefusalsArePlainEnglish(t *testing.T) {
	c, id := typedRun(t)
	ref := "SampleTypedCTRL/GGIO4.AnOut1"
	if _, err := c.SelectTyped(TypedCommand{IEDID: id, Ref: ref, Value: 1.5}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3200 * time.Millisecond)
	_, err := c.OperateTyped(TypedCommand{IEDID: id, Ref: ref, Value: 1.5})
	var studio *apperr.Error
	if !errors.As(err, &studio) || studio.Message != "Select expired after 3 s. Press Select again." {
		t.Fatalf("expired select: %v", err)
	}
	for value, want := range map[any]string{int64(11): "between 0 and 10", "x": "whole number"} {
		_, err := c.OperateTyped(TypedCommand{IEDID: id, Ref: "SampleTypedCTRL/GGIO1.ISCSO1", Value: value})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("INC %v: %v", value, err)
		}
	}
	if err := c.SetControlChecks(id, "SampleTypedCTRL/GGIO1.ISCSO1", pyjson.NewObject().Set("interlock_blocked", true)); err != nil {
		t.Fatal(err)
	}
	_, err = c.OperateTyped(TypedCommand{IEDID: id, Ref: "SampleTypedCTRL/GGIO1.ISCSO1", Value: int64(3),
		Options: CommandOptions{InterlockCheck: true}})
	if err == nil || !strings.Contains(err.Error(), "blocked by interlocking") {
		t.Fatalf("interlock: %v", err)
	}
	if _, err := c.OperateTyped(TypedCommand{IEDID: "nope", Ref: ref, Value: 1.0}); err == nil || err.Error() != "Unknown IED" {
		t.Fatalf("unknown IED: %v", err)
	}
	if _, err := c.OperateTyped(TypedCommand{IEDID: id, Ref: "SampleTypedCTRL/GGIO1.Nope", Value: 1.0}); err == nil ||
		err.Error() != "Unknown control object" {
		t.Fatalf("unknown control: %v", err)
	}
}
