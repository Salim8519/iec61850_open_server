package native

import (
	"time"

	"studio-engine/internal/pyjson"
)

// Control texts from tools/control_client.py.
const (
	ControlMissing  = "The control program (studio_control.exe) is not built yet; rebuild the native programs"
	ControlOutdated = "The control program is outdated; rebuild the native programs"
	// MuxCapability marks a studio_control that serves canvas operations in parallel per
	// endpoint, matched by request id (ADR proposal in streams/c-runtime.md).
	MuxCapability = "control_mux_v1"
)

// Control is a typed client for studio_control.exe.
type Control struct{ *Worker }

// StartControl launches studio_control.exe (no event consumer, as control_client.py).
func StartControl(executable string) (*Control, error) {
	worker, err := Start(executable, nil)
	if err != nil {
		return nil, err
	}
	return &Control{worker}, nil
}

// Hello returns {capabilities}.
func (c *Control) Hello() (*pyjson.Object, error) {
	return objectResult(c.Request("hello", nil, DefaultTimeout))
}

// Operate is the Tools command (IED Explorer, gateway): fields in control_client.py order
// (ip, port, ref, value, or_ident, or_cat, interlock_check, synchro_check, test); 30 s.
func (c *Control) Operate(fields *pyjson.Object) (*pyjson.Object, error) {
	return objectResult(c.Request("operate", fields, 30*time.Second))
}

// CanvasOperate sends one canvas command with exactly what `studio_probe IP PORT control
// REF true|false` sends on the wire (no originator set, interlock/synchro/test off, SBO or
// enhanced by the device's ctlModel, up to 100x20 ms for CommandTermination). keep is how long
// the association stays open for the next command to the same endpoint: 0 closes it after
// every command like the probe process did. The result is the probe's {"ok": bool}.
func (c *Control) CanvasOperate(ip string, port int64, ref string, value bool, keep time.Duration,
	timeout time.Duration) (*pyjson.Object, error) {
	fields := pyjson.NewObject().Set("ip", ip).Set("port", port).Set("ref", ref).Set("value", value).
		Set("keep_ms", int64(keep/time.Millisecond))
	return objectResult(c.Request("canvas_operate", fields, timeout))
}
