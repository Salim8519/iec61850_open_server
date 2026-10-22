package control

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// TypedOperator is what IED Explorer needs for typed Select, Operate and Cancel (ADR-0104).
type TypedOperator interface {
	Typed(op string, fields *pyjson.Object) (*pyjson.Object, error)
}

// Typed sends one typed step (typed_select, typed_operate or typed_cancel) through the warm
// studio_control worker, one command at a time like Operate. A worker without the
// typed_commands capability is outdated.
func (c *Client) Typed(op string, fields *pyjson.Object) (*pyjson.Object, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	process, err := c.worker()
	if err != nil {
		return nil, err
	}
	hello, err := process.Request("hello", OperateTimeout, nil)
	if err != nil {
		return nil, err
	}
	if !hasCapability(hello, "typed_commands") {
		return nil, apperr.RuntimeError("The control program is outdated; rebuild the native programs")
	}
	answer, err := process.Request(op, OperateTimeout, fields)
	if err != nil {
		return nil, err
	}
	result, ok := answer.(*pyjson.Object)
	if !ok {
		return pyjson.NewObject(), nil
	}
	return result, nil
}
