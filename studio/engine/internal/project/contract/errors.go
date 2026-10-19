// Package contract validates browser-editable project geometry exactly as the pydantic
// 2.10.6 models of contracts/project.py do: the same accepted inputs (lax mode), the same
// model_dump() output with every default materialised, and the same error list.
package contract

import "studio-engine/internal/pyjson"

// ContractError is pydantic's ValidationError. Details reproduce
// errors(include_context=False, include_input=False): ordered objects with type, loc, msg
// and url. The HTTP layer answers 400 {"details": [...], "error": "Invalid request"}.
type ContractError struct {
	Details []any
}

func (e *ContractError) Error() string { return "Invalid request" }

const errorURL = "https://errors.pydantic.dev/2.10/v/"

// loc is a pydantic error location: field names and list indexes.
type loc []any

func (l loc) with(part any) loc {
	next := make(loc, len(l), len(l)+1)
	copy(next, l)
	return append(next, part)
}

// collector gathers errors in validation order.
type collector struct {
	details []any
}

func (c *collector) add(kind string, at loc, message string) {
	item := pyjson.NewObject()
	item.Set("type", kind)
	item.Set("loc", []any(append(loc{}, at...)))
	item.Set("msg", message)
	item.Set("url", errorURL+kind)
	c.details = append(c.details, item)
}

// Validator checks one input value at a location; ok is false when it added errors.
type Validator func(c *collector, at loc, input any) (any, bool)
