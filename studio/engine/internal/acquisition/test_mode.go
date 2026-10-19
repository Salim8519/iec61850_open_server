package acquisition

import (
	"strings"

	"studio-engine/internal/acquisition/modes"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// RealRefusal is set_test_mode's refusal for anything but a simulated IED.
const RealRefusal = "Mode and simulation can only be set on IEDs that ScadaBench simulates. " +
	"ScadaBench never changes a real device."

// TestModes applies saved test modes to ScadaBench's simulated IEDs, never to real devices
// (simulation/test_mode_apply.py).
type TestModes struct{ deps Deps }

// NewTestModes returns the service.
func NewTestModes(deps Deps) *TestModes { return &TestModes{deps: deps} }

func (t *TestModes) write(ied *pyjson.Object) (int, error) {
	id := pyv.GetStr(ied, "id")
	if !pyv.Equal(pyv.Get(ied, "role"), "simulated") || !t.deps.Samples.Running(id) {
		return 0, nil
	}
	view := modeView(ied)
	current := pyjson.NewObject()
	if t.deps.Values.Has(id) {
		for _, signal := range pyv.Objects(pyv.GetList(view, "signals")) {
			ref := pyv.GetStr(signal, "ref")
			if !strings.HasSuffix(ref, ".Mod.stVal") {
				continue
			}
			if value, _, ok := t.deps.Values.Get(id, ref); ok {
				current.Set(ref, value)
			}
		}
	}
	settings := pyv.GetObj(ied, "test_mode")
	if !pyv.Truthy(settings) {
		settings, _ = modes.ValidateTestMode(nil)
	}
	writes := modes.ModeWrites(view, settings, current)
	if len(writes) > 0 {
		values := make([]RefValue, len(writes))
		for index, item := range writes {
			values[index] = RefValue{Ref: item.Ref, Value: item.Value}
		}
		if err := t.deps.Samples.Apply(id, values, "test_mode"); err != nil {
			return 0, err
		}
	}
	return len(writes), nil
}

// ApplyAll writes, at Run start, every simulated IED's saved modes that differ from normal
// service; only limits it to some IEDs when not nil.
func (t *TestModes) ApplyAll(only map[string]bool) error {
	for _, ied := range pyv.Objects(pyv.GetList(t.deps.Projects.Current(), "ieds")) {
		if only != nil && !only[pyv.GetStr(ied, "id")] {
			continue
		}
		settings := pyv.OrObj(ied, "test_mode")
		if pyv.Truthy(pyv.Get(settings, "devices")) || pyv.Truthy(pyv.Get(settings, "simulation")) {
			if _, err := t.write(ied); err != nil {
				return err
			}
		}
	}
	return nil
}

// Set saves the modes of one simulated IED and applies them at once when it is running:
// PUT /ieds/<id>/test-mode answers {"test_mode": settings, "written": n}.
func (t *TestModes) Set(iedID string, value any) (*pyjson.Object, error) {
	ied, err := t.deps.Projects.FindIED(iedID)
	if err != nil {
		return nil, err
	}
	if !pyv.Equal(pyv.Get(ied, "role"), "simulated") {
		return nil, apperr.New(RealRefusal)
	}
	settings, err := modes.ValidateTestMode(value)
	if err != nil {
		return nil, apperr.New(pyv.PyStr(pyv.Get(ied, "name")) + ": " + err.Error())
	}
	saved, err := t.deps.Projects.SetTestMode(iedID, settings)
	if err != nil {
		return nil, err
	}
	written := 0
	if state := t.deps.Runtime.State(); state == "running" || state == "paused" {
		if written, err = t.write(saved); err != nil {
			return nil, err
		}
	}
	return pyv.O("test_mode", settings, "written", written), nil
}
