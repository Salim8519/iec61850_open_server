package native

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// IedRequired are the capabilities a studio_ied start reply must list
// (runtime/worker_settings.py REQUIRED_CAPABILITIES).
var IedRequired = []string{"versioned_updates", "atomic_control_feedback", "control_selection_events",
	"standard_controls", "goose_aware", "goose_bus"}

// StaleIed is the refusal text for an outdated studio_ied.exe.
const StaleIed = "Native engine is outdated. Use the rebuilt portable application or rebuild the native worker before Run"

// Ied is a typed client for studio_ied.exe: every call is one request with the 10 s default.
type Ied struct{ *Worker }

// StartIed launches studio_ied.exe.
func StartIed(executable string, onEvent EventFunc) (*Ied, error) {
	worker, err := Start(executable, onEvent)
	if err != nil {
		return nil, err
	}
	return &Ied{worker}, nil
}

// StartModel sends "start"; fields are in supervisor/worker_settings order
// (cfg, ip, port, controls, edition, goose). The reply's capabilities are checked.
func (c *Ied) StartModel(fields *pyjson.Object) (*pyjson.Object, error) {
	result, err := c.Request("start", fields, DefaultTimeout)
	if err != nil {
		return nil, err
	}
	started, _ := result.(*pyjson.Object)
	if started == nil {
		started = pyjson.NewObject()
	}
	if !HasCapabilities(started, IedRequired) {
		return nil, apperr.ValueError(StaleIed)
	}
	return started, nil
}

// Update writes one attribute and returns the value read back.
func (c *Ied) Update(ref string, value any) (any, error) {
	return c.Request("update", pyjson.NewObject().Set("ref", ref).Set("value", value), DefaultTimeout)
}

// UpdateBatch writes up to 256 {ref, value} items and returns {ref: value read back}.
func (c *Ied) UpdateBatch(updates []any) (*pyjson.Object, error) {
	result, err := c.Request("update_batch", pyjson.NewObject().Set("updates", updates), DefaultTimeout)
	return objectResult(result, err)
}

// UpdateGuarded writes when the guard's revision still holds: {values, revision}.
func (c *Ied) UpdateGuarded(updates []any, guard *pyjson.Object) (*pyjson.Object, error) {
	result, err := c.Request("update_guarded", pyjson.NewObject().Set("updates", updates).Set("guard", guard), DefaultTimeout)
	return objectResult(result, err)
}

// Read returns one attribute's value.
func (c *Ied) Read(ref string) (any, error) {
	return c.Request("read", pyjson.NewObject().Set("ref", ref), DefaultTimeout)
}

// ReadVersioned returns {value, revision}.
func (c *Ied) ReadVersioned(ref string) (*pyjson.Object, error) {
	result, err := c.Request("read_versioned", pyjson.NewObject().Set("ref", ref), DefaultTimeout)
	return objectResult(result, err)
}

func objectResult(result any, err error) (*pyjson.Object, error) {
	if err != nil {
		return nil, err
	}
	object, _ := result.(*pyjson.Object)
	if object == nil {
		object = pyjson.NewObject()
	}
	return object, nil
}

// HasCapabilities reports whether reply["capabilities"] lists every required name.
func HasCapabilities(reply *pyjson.Object, required []string) bool {
	listed, _ := reply.Get("capabilities")
	items, _ := listed.([]any)
	have := map[string]bool{}
	for _, item := range items {
		if name, ok := item.(string); ok {
			have[name] = true
		}
	}
	for _, name := range required {
		if !have[name] {
			return false
		}
	}
	return true
}
