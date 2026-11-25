package realcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
	"studio-engine/internal/tools/control"
	"studio-engine/internal/tools/nativetest"
	"studio-engine/internal/tools/worker"
	"studio-engine/internal/typedcmd"
)

// Loopback address of the Devices command wire test (ADR-0149); ports >= 20000.
const wireIP, wirePort = "127.67.0.47", int64(20247)

type anySchema struct{}

func (anySchema) Validate([]byte, string) (scl.SchemaVerdict, error) {
	return scl.SchemaVerdict{Status: "valid"}, nil
}

// deviceSession reads the simulated IED as the live session reads a real one.
type deviceSession struct{ device worker.Process }

func (d deviceSession) State() string { return "live" }
func (d deviceSession) ReadOne(ref, _ string) (any, bool, error) {
	value, err := d.device.Request("read", 10*time.Second, pyjson.NewObject().Set("ref", ref))
	return value, err == nil && value != nil, err
}

// wireRig is the SampleTyped IED in studio_ied, monitored as a real IED with Allow commands on, and
// the Devices command service over the real studio_control client.
func wireRig(t *testing.T, local string) (*Service, worker.Process, *[]string) {
	t.Helper()
	source, err := os.ReadFile(filepath.Join(nativetest.StudioDir(), "tests", "fixtures", "typed_controls", "SampleTyped.scd"))
	if err != nil {
		t.Fatal(err)
	}
	models, err := (&scl.Importer{Validator: anySchema{}}).Import(source, "SampleTyped.scd", scl.SelectAll)
	if err != nil {
		t.Fatal(err)
	}
	extended, err := scl.ExtendedControls(source, "SampleTyped")
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := models[0].Get("cfg")
	path := filepath.Join(t.TempDir(), "typed.cfg")
	if err := os.WriteFile(path, []byte(cfg.(string)), 0o644); err != nil {
		t.Fatal(err)
	}
	device, err := worker.Spawn(nativetest.Program(t, "studio_ied.exe"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(device.Close)
	if _, err := device.Request("start", 30*time.Second, pyjson.NewObject().Set("cfg", path).Set("ip", wireIP).
		Set("port", wirePort).Set("controls", []any{}).Set("edition", int64(2)).Set("extended_controls", extended)); err != nil {
		t.Fatal(err)
	}
	client := control.NewClient(nativetest.Program(t, "studio_control.exe"), nil)
	t.Cleanup(client.Close)
	live := pyjson.NewObject().Set("ip", wireIP).Set("port", wirePort).Set("allow_commands", true)
	if local != "" {
		live.Set("local", local)
	}
	ied := pyjson.NewObject().Set("id", "w1").Set("name", "SampleTyped").Set("role", "real").Set("live", live).
		Set("controls", []any{})
	var events []string
	service := &Service{FindIED: func(string) (*pyjson.Object, error) { return ied, nil },
		Sessions: func(string) (Session, error) { return deviceSession{device}, nil }, Control: client,
		Extended: func(*pyjson.Object) []any { return extended },
		Emit:     func(kind string, _ *pyjson.Object) { events = append(events, kind) }}
	return service, device, &events
}

func send(t *testing.T, s *Service, step Step) *pyjson.Object {
	t.Helper()
	step.Ref = "SampleTypedCTRL/" + step.Ref
	step.Confirmed = true
	answer, err := s.Command("w1", step)
	if err != nil {
		t.Fatalf("%s %s: %v", step.Action, step.Ref, err)
	}
	return answer
}

// Every command class in every control model, one click: the device operates it and the feedback
// shows the value (ENC by label, ISC Raise one position up, BSC/BAC Raise, INC, APC).
func TestDevicesCommandsEveryClassOneClick(t *testing.T) {
	s, _, _ := wireRig(t, "")
	for _, n := range []string{"1", "2", "3", "4"} {
		for _, c := range []struct {
			ref   string
			value any
		}{{"GGIO" + n + ".Mod", "test"}, {"GGIO" + n + ".ISCSO1", int64(7)}, {"GGIO" + n + ".AnOut1", 42.5},
			{"ATCC" + n + ".TapChg", "higher"}, {"ATCC" + n + ".TapSet", int64(9)}, {"ATCC" + n + ".TapSet", "higher"},
			{"ATCC" + n + ".BndSet", "higher"}} {
			answer := send(t, s, Step{Action: "operate", Ref: c.ref, Value: c.value})
			if get(answer, "ok") != true {
				t.Fatalf("%s %v: %v", c.ref, c.value, get(answer, "summary"))
			}
			if observed, _ := get(answer, "observed").(string); get(answer, "changed") != true && !strings.Contains(observed, ".mxVal") {
				t.Errorf("%s %v feedback: %v", c.ref, c.value, observed)
			}
		}
	}
}

// Step by step through a chosen card: Select reserves (SBO) or answers that nothing is needed
// (direct), Operate uses the selection, Cancel gives a selection back.
func TestDevicesCommandsStepByStepThroughACard(t *testing.T) {
	s, _, events := wireRig(t, "127.0.0.1")
	for _, n := range []string{"2", "4"} {
		selected := send(t, s, Step{Action: "select", Ref: "GGIO" + n + ".ISCSO1", Value: int64(3)})
		if get(selected, "ok") != true || get(selected, "expires_ms") != int64(2700) {
			t.Fatalf("select GGIO%s: %v", n, selected)
		}
		operated := send(t, s, Step{Action: "operate", Ref: "GGIO" + n + ".ISCSO1", Value: int64(3), Selected: true})
		if get(operated, "ok") != true || get(operated, "changed") != true {
			t.Fatalf("operate GGIO%s: %v", n, operated)
		}
	}
	notSelected := send(t, s, Step{Action: "operate", Ref: "GGIO4.ISCSO1", Value: int64(4), Selected: true})
	if get(notSelected, "ok") != false || !strings.Contains(get(notSelected, "summary").(string), "Select first") {
		t.Fatalf("operate without a select in step mode: %v", notSelected)
	}
	send(t, s, Step{Action: "select", Ref: "GGIO2.Mod", Value: "on"})
	if canceled := send(t, s, Step{Action: "cancel", Ref: "GGIO2.Mod"}); get(canceled, "ok") != true {
		t.Fatalf("cancel: %v", canceled)
	}
	direct := send(t, s, Step{Action: "select", Ref: "GGIO1.ISCSO1", Value: int64(2)})
	if get(direct, "ok") != true || !strings.Contains(get(direct, "summary").(string), "operates directly") {
		t.Fatalf("select of a direct control: %v", direct)
	}
	if !strings.Contains(strings.Join(*events, ","), "real_command_canceled") {
		t.Fatalf("events %v", *events)
	}
}

// The device's interlock block comes back in words; without the check the same command passes.
func TestDevicesInterlockRefusalInWords(t *testing.T) {
	s, device, _ := wireRig(t, "")
	if _, err := device.Request("control_checks", 10*time.Second, pyjson.NewObject().
		Set("ref", "SampleTypedCTRL/GGIO1.ISCSO1").Set("interlock_blocked", true)); err != nil {
		t.Fatal(err)
	}
	checked := typedcmd.Options{InterlockCheck: true, OrCat: 2}
	blocked := send(t, s, Step{Action: "operate", Ref: "GGIO1.ISCSO1", Value: int64(2), Options: checked})
	if summary, _ := get(blocked, "summary").(string); get(blocked, "ok") != false ||
		!strings.Contains(summary, "interlocking does not allow") || !strings.Contains(summary, "Device answer:") {
		t.Fatalf("interlock refusal: %v", blocked)
	}
	if free := send(t, s, Step{Action: "operate", Ref: "GGIO1.ISCSO1", Value: int64(2)}); get(free, "ok") != true {
		t.Fatalf("without interlock check: %v", free)
	}
}
