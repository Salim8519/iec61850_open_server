package control

import (
	"os"
	"sync"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/worker"
)

// OperateTimeout is the 30 s a device may take to select, operate and terminate.
const OperateTimeout = 30 * time.Second

// Operator is what Explorer and the gateway need from the control client.
type Operator interface {
	Operate(host string, port any, reference string, value any, options any) (*pyjson.Object, error)
}

// Client keeps one warm studio_control worker; commands run one at a time (the native
// worker is single-threaded anyway, and a SCADA command waits in line like in Python).
type Client struct {
	executable string
	spawn      worker.Spawner
	mu         sync.Mutex
	process    worker.Process
}

// NewClient returns a client for executable (studio_control.exe beside the IED worker).
func NewClient(executable string, spawn worker.Spawner) *Client {
	if spawn == nil {
		spawn = worker.Spawn
	}
	return &Client{executable: executable, spawn: spawn}
}

func (c *Client) worker() (worker.Process, error) {
	if c.process != nil && c.process.Alive() {
		return c.process, nil
	}
	if _, err := os.Stat(c.executable); err != nil {
		return nil, apperr.RuntimeError("The control program (studio_control.exe) is not built yet; rebuild the native programs")
	}
	process, err := c.spawn(c.executable, nil)
	if err != nil {
		return nil, err
	}
	hello, err := process.Request("hello", worker.DefaultTimeout, nil)
	if err != nil {
		process.Close()
		return nil, err
	}
	if !hasCapability(hello, "control_operate") {
		process.Close()
		return nil, apperr.RuntimeError("The control program is outdated; rebuild the native programs")
	}
	c.process = process
	return process, nil
}

// Operate returns the device's answer: ok, model, selected, operated, terminated, add_cause_text, error.
func (c *Client) Operate(host string, port any, reference string, value any, options any) (*pyjson.Object, error) {
	settings, err := Options(options)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	process, err := c.worker()
	if err != nil {
		return nil, err
	}
	checked, err := Value(value)
	if err != nil {
		return nil, err
	}
	fields := pyjson.NewObject().Set("ip", host).Set("port", port).Set("ref", reference).
		Set("value", checked).Set("or_ident", "ScadaBench")
	for _, key := range settings.Keys() {
		item, _ := settings.Get(key)
		fields.Set(key, item)
	}
	answer, err := process.Request("operate", OperateTimeout, fields)
	if err != nil {
		return nil, err
	}
	result, ok := answer.(*pyjson.Object)
	if !ok {
		return pyjson.NewObject(), nil
	}
	return result, nil
}

// Close stops the warm worker.
func (c *Client) Close() {
	c.mu.Lock()
	process := c.process
	c.process = nil
	c.mu.Unlock()
	if process != nil {
		process.Close()
	}
}

func hasCapability(hello any, name string) bool {
	object, ok := hello.(*pyjson.Object)
	if !ok {
		return false
	}
	list, _ := object.Get("capabilities")
	items, _ := list.([]any)
	for _, item := range items {
		if item == name {
			return true
		}
	}
	return false
}

// HasCapability reports whether a hello answer lists name (shared by the other workers).
func HasCapability(hello any, name string) bool { return hasCapability(hello, name) }
