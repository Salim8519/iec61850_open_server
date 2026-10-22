package runtime

import (
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/native"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/typedcmd"
)

// CommandOptions are the IEC 61850 command options (ADR-0104). OrCat 0 means "not set" and
// sends 3 (remote control); OrIdent "" sends "ScadaBench".
type CommandOptions = typedcmd.Options

// TypedCommand is one typed command to a simulated IED of the running project.
type TypedCommand struct {
	IEDID   string         `json:"ied_id"`
	Ref     string         `json:"ref"`   // control object, e.g. LD/CSWI1.Pos or LD/ATCC1.TapChg
	Value   any            `json:"value"` // bool | int | float | enum label | "higher"/"lower"/"stop"
	Options CommandOptions `json:"options"`
}

// SelectResult is what a Select reserved: the model and how long the selection lasts.
type SelectResult struct {
	Model     string `json:"model"`      // direct-normal|direct-enhanced|sbo-normal|sbo-enhanced
	ExpiresMs int    `json:"expires_ms"` // 0 for direct models
}

type typedTarget struct {
	control typedcmd.Control
	host    string
	port    int64
	current any
}

// typedControl finds the running IED and describes the control (SPC/DPC from "controls",
// the other classes from the stored SCL). Only simulated IEDs of this run are commanded here.
func (c *Coordinator) typedControl(iedID, ref string) (*typedTarget, error) {
	ied := findIed(c.opt.Project.Current(), iedID)
	if ied == nil {
		return nil, apperr.Coded("Unknown IED", "not_found")
	}
	c.mu.Lock()
	item, state := c.runs[iedID], c.iedStates[iedID]
	c.mu.Unlock()
	if item == nil || item.worker == nil || state != "running" {
		return nil, apperr.ValueError("IED must be running to operate equipment")
	}
	target := &typedTarget{host: item.host, port: item.port}
	if entry := item.model.control(ref); entry != nil {
		target.control = typedcmd.FromStandard(entry)
	} else {
		for _, raw := range extendedControlsOf(ied) {
			if entry, _ := raw.(*pyjson.Object); entry != nil && pyval.Str(entry, "ref") == ref {
				target.control = typedcmd.FromExtended(entry)
				target.current, _ = item.table.get(pyval.Str(entry, "status"))
			}
		}
	}
	if target.control.Ref == "" {
		return nil, apperr.ValueError("Unknown control object")
	}
	if target.control.Model == 0 {
		return nil, apperr.ValueError("This object is status-only; it cannot be operated")
	}
	return target, nil
}

func (c *Coordinator) typedProcess() (*native.Control, error) {
	process, err := c.commands.client()
	if err != nil {
		return nil, err
	}
	if !c.commands.typedOK {
		return nil, apperr.RuntimeError(native.ControlOutdated)
	}
	return process, nil
}

func typedFields(target *typedTarget, options CommandOptions) *pyjson.Object {
	if options.OrCat == 0 {
		options.OrCat = typedcmd.DefaultOrCat
	}
	return options.Fields(pyjson.NewObject().Set("ip", target.host).Set("port", target.port).
		Set("ref", target.control.Ref))
}

// SelectTyped reserves an SBO control (with its value for enhanced security). Direct models
// need no select and answer ExpiresMs 0.
func (c *Coordinator) SelectTyped(cmd TypedCommand) (SelectResult, error) {
	target, err := c.typedControl(cmd.IEDID, cmd.Ref)
	if err != nil {
		return SelectResult{}, err
	}
	if err := cmd.Options.Check(); err != nil {
		return SelectResult{}, err
	}
	fields := typedFields(target, cmd.Options)
	if cmd.Value != nil || target.control.Model == 4 {
		kind, wire, err := typedcmd.Wire(target.control, cmd.Value, target.current)
		if err != nil {
			return SelectResult{}, err
		}
		fields.Set("value", wire).Set("kind", kind)
	}
	process, err := c.typedProcess()
	if err != nil {
		return SelectResult{}, err
	}
	c.access.markOwn(cmd.IEDID, cmd.Ref)
	result, err := process.Typed("typed_select", fields)
	if err != nil {
		return SelectResult{}, err
	}
	if err := typedcmd.Outcome(result); err != nil {
		return SelectResult{}, err
	}
	expires, _ := pyval.Get(result, "expires_ms").(int64)
	if expires > 0 {
		c.commands.selects.remember(cmd.IEDID, cmd.Ref, expires, time.Now())
	}
	return SelectResult{Model: typedcmd.ModelName(target.control.Model), ExpiresMs: int(expires)}, nil
}

// OperateTyped operates with a typed value. After SelectTyped it uses that selection (and
// refuses once it expired); without one it selects and operates in one step, as Operate does.
func (c *Coordinator) OperateTyped(cmd TypedCommand) (*pyjson.Object, error) {
	target, err := c.typedControl(cmd.IEDID, cmd.Ref)
	if err != nil {
		return nil, err
	}
	if err := cmd.Options.Check(); err != nil {
		return nil, err
	}
	kind, wire, err := typedcmd.Wire(target.control, cmd.Value, target.current)
	if err != nil {
		return nil, err
	}
	process, err := c.typedProcess()
	if err != nil {
		return nil, err
	}
	selected, expired, timeout := c.commands.selects.take(cmd.IEDID, cmd.Ref, time.Now())
	if expired {
		return nil, typedcmd.SelectExpired(timeout)
	}
	fields := typedFields(target, cmd.Options).Set("value", wire).Set("kind", kind).Set("require_select", selected)
	c.access.markOwn(cmd.IEDID, cmd.Ref)
	result, err := process.Typed("typed_operate", fields)
	if err != nil {
		return nil, err
	}
	if err := typedcmd.Outcome(result); err != nil {
		return nil, err
	}
	return result, nil
}

// CancelTyped ends a selection before it is operated. A Cancel the device refuses keeps the
// selection (change 0176): the device still holds it until its time runs out.
func (c *Coordinator) CancelTyped(iedID, ref string) error {
	target, err := c.typedControl(iedID, ref)
	if err != nil {
		return err
	}
	process, err := c.typedProcess()
	if err != nil {
		return err
	}
	result, err := process.Typed("typed_cancel", typedFields(target, CommandOptions{}))
	if err != nil {
		return err
	}
	if err := cancelOutcome(result, ref); err != nil {
		return err
	}
	c.commands.selects.forget(iedID, ref)
	return nil
}

// noSuchObject is libiec61850's IED_ERROR_OBJECT_DOES_NOT_EXIST.
const noSuchObject = 22

// cancelOutcome is the typed outcome, in plain words when the device's model has no Cancel.
func cancelOutcome(result *pyjson.Object, ref string) error {
	if code, _ := pyval.Get(result, "client_error").(int64); code == noSuchObject && pyval.Get(result, "ok") != true {
		return apperr.Coded("This device's model has no Cancel for "+ref+", so the device keeps the selection "+
			"until its time runs out. Press Operate to send the command, or wait.", "conflict")
	}
	return typedcmd.Outcome(result)
}
