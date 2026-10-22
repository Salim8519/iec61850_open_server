package control

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
	"studio-engine/internal/tools/nativetest"
	"studio-engine/internal/tools/worker"
)

// Loopback address of the typed-command wire tests (ADR-0104); ports >= 20000.
const typedIP, typedPort = "127.67.0.41", int64(20241)

type validSchema struct{}

func (validSchema) Validate([]byte, string) (scl.SchemaVerdict, error) {
	return scl.SchemaVerdict{Status: "valid"}, nil
}

// rig is the SampleTyped IED in studio_ied plus a typed control client.
type rig struct {
	device worker.Process
	client *Client
	mu     sync.Mutex
	events []*pyjson.Object
}

func sampleSCL(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(nativetest.StudioDir(), "tests", "fixtures", "typed_controls", "SampleTyped.scd"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func startRig(t *testing.T) *rig {
	t.Helper()
	ied := nativetest.Program(t, "studio_ied.exe")
	controlExe := nativetest.Program(t, "studio_control.exe")
	source := sampleSCL(t)
	models, err := (&scl.Importer{Validator: validSchema{}}).Import(source, "SampleTyped.scd", scl.SelectAll)
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
	r := &rig{}
	r.device, err = worker.Spawn(ied, func(event *pyjson.Object) {
		r.mu.Lock()
		r.events = append(r.events, event)
		r.mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.device.Close)
	fields := pyjson.NewObject().Set("cfg", path).Set("ip", typedIP).Set("port", typedPort).
		Set("controls", []any{}).Set("edition", int64(2)).Set("extended_controls", extended)
	started, err := r.device.Request("start", 30*time.Second, fields)
	if err != nil {
		t.Fatal(err)
	}
	if !HasCapability(started, "typed_controls") {
		t.Fatalf("worker must advertise typed_controls: %v", started)
	}
	r.client = NewClient(controlExe, nil)
	t.Cleanup(r.client.Close)
	return r
}

func (r *rig) step(t *testing.T, op, ref, kind string, value any, extra ...any) *pyjson.Object {
	t.Helper()
	fields := pyjson.NewObject().Set("ip", typedIP).Set("port", typedPort).Set("ref", "SampleTypedCTRL/"+ref).
		Set("or_ident", "rig-test").Set("or_cat", int64(2))
	if value != nil {
		fields.Set("value", value).Set("kind", kind)
	}
	for i := 0; i+1 < len(extra); i += 2 {
		fields.Set(extra[i].(string), extra[i+1])
	}
	result, err := r.client.Typed(op, fields)
	if err != nil {
		t.Fatalf("%s %s: %v", op, ref, err)
	}
	return result
}

func (r *rig) read(t *testing.T, ref string) any {
	t.Helper()
	value, err := r.device.Request("read", 10*time.Second, pyjson.NewObject().Set("ref", "SampleTypedCTRL/"+ref))
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func (r *rig) checks(t *testing.T, ref string, settings ...any) {
	t.Helper()
	fields := pyjson.NewObject().Set("ref", "SampleTypedCTRL/"+ref)
	for i := 0; i+1 < len(settings); i += 2 {
		fields.Set(settings[i].(string), settings[i+1])
	}
	if _, err := r.device.Request("control_checks", 10*time.Second, fields); err != nil {
		t.Fatal(err)
	}
}

func (r *rig) lastApplied(ref string) *pyjson.Object {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.events) - 1; i >= 0; i-- {
		kind, _ := r.events[i].Get("event")
		at, _ := r.events[i].Get("ref")
		if kind == "control_applied" && at == "SampleTypedCTRL/"+ref {
			return r.events[i]
		}
	}
	return nil
}

func (r *rig) fields(ref string) *pyjson.Object {
	return pyjson.NewObject().Set("ip", typedIP).Set("port", typedPort).Set("ref", "SampleTypedCTRL/"+ref)
}

func field(o *pyjson.Object, key string) any {
	if o == nil {
		return nil
	}
	value, _ := o.Get(key)
	return value
}

func number(value any) float64 {
	switch v := value.(type) {
	case int64:
		return float64(v)
	case float64:
		return v
	}
	return -9999
}
