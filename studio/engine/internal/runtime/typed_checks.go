package runtime

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/native"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/scl"
)

// extendedControls is the Go-only start key "extended_controls" of one IED (ADR-0104): the
// typed controls read from its stored SCL. An IED whose SCL cannot be read again starts with
// its SPC/DPC controls only; the import already accepted that SCL, so this is not expected.
func (m *iedModel) extendedControls() []any {
	controls, err := scl.CachedExtendedControls(m.id, m.name, pyval.Str(m.raw, "source"))
	if err != nil {
		return nil
	}
	return controls
}

// writableDeadbands is the Go-only start key "writable_deadbands" (ADR-0118): the db/zeroDb
// references a client may change, those the SCD marks valKind Set or leaves unmarked.
func (m *iedModel) writableDeadbands() []any {
	refs, err := scl.CachedDeadbandWrites(m.id, m.name, pyval.Str(m.raw, "source"))
	if err != nil {
		return nil
	}
	list := make([]any, len(refs))
	for i, ref := range refs {
		list[i] = ref
	}
	return list
}

// startRequest is the worker start request; "extended_controls" and "writable_deadbands" are
// added only when the IED has them, so every other start request stays exactly what it was.
func startRequest(item *running, cfg string, point endpoint, goose *pyjson.Object) *pyjson.Object {
	request := pyval.Obj("cfg", cfg, "ip", point.host, "port", point.port, "controls", item.model.controls(),
		"edition", pyval.Get(item.ied, "edition"), "goose", goose)
	if extended := item.model.extendedControls(); len(extended) > 0 {
		request.Set("extended_controls", extended)
	}
	if deadbands := item.model.writableDeadbands(); len(deadbands) > 0 {
		request.Set("writable_deadbands", deadbands)
	}
	return request
}

// TypedControls lists what an IED can be commanded with: {"controls", "extended_controls"}.
func (c *Coordinator) TypedControls(iedID string) (*pyjson.Object, error) {
	ied := findIed(c.opt.Project.Current(), iedID)
	if ied == nil {
		return nil, apperr.Coded("Unknown IED", "not_found")
	}
	extended, err := scl.CachedExtendedControls(iedID, pyval.Str(ied, "name"), pyval.Str(ied, "source"))
	if err != nil {
		return nil, err
	}
	return pyval.Obj("controls", pyval.List(ied, "controls"), "extended_controls", extended), nil
}

// SetControlChecks sets the simulated blocks of one typed control of a running IED:
// {"blocked"?, "interlock_blocked"?, "synchro_blocked"?} (booleans). A command that asks for
// interlock or synchro check is then refused with that AddCause.
func (c *Coordinator) SetControlChecks(iedID, ref string, checks *pyjson.Object) error {
	target, err := c.typedControl(iedID, ref)
	if err != nil {
		return err
	}
	if target.control.CDC == "SPC" || target.control.CDC == "DPC" {
		return apperr.ValueError("Simulated checks are for typed controls (INC, ENC, APC, BAC, BSC, ISC)")
	}
	fields := pyjson.NewObject().Set("ref", ref)
	for _, key := range []string{"blocked", "interlock_blocked", "synchro_blocked"} {
		if value, found := checks.Get(key); found {
			flag, ok := value.(bool)
			if !ok {
				return apperr.ValueError(key + " is true or false")
			}
			fields.Set(key, flag)
		}
	}
	c.mu.Lock()
	item := c.runs[iedID]
	c.mu.Unlock()
	if item == nil || item.worker == nil {
		return apperr.ValueError("IED is not running")
	}
	_, err = item.worker.Request("control_checks", native.DefaultTimeout, fields)
	return err
}
