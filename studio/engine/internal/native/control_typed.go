package native

import (
	"time"

	"studio-engine/internal/pyjson"
)

// TypedCapability marks a studio_control that serves typed Select, Operate and Cancel
// (typed_select, typed_operate, typed_cancel; ADR-0104).
const TypedCapability = "typed_commands"

// TypedTimeout covers select, operate and the 5 s wait for CommandTermination.
const TypedTimeout = 30 * time.Second

// Typed sends one typed step; op is typed_select, typed_operate or typed_cancel. The reply is
// {ref, model, selected, operated, canceled, terminated, add_cause, add_cause_text, expires_ms, ok, error?}.
func (c *Control) Typed(op string, fields *pyjson.Object) (*pyjson.Object, error) {
	return objectResult(c.call(op, fields, TypedTimeout))
}
