// Package goosedom holds the pure GOOSE rules of the live acquisition stream: what to
// watch and LGOS supervision, where GOOSE goes (inside or shared), binding subscriber
// inputs, the declared link map joined with the listener's view, and the adapter list
// ordering. Ports of domain/goose_supervision.py, goose_reach.py, goose_inputs.py,
// goose_link_map.py, goose_link_live.py and networking/adapters.py (normalize only).
//
// Every function takes pyjson trees and returns fresh pyjson values; inputs are never
// modified. Python mutates some of these dicts in place at its call sites; Go callers
// own the returned objects and store them where Python would have mutated.
package goosedom

import (
	"math"
	"math/big"
	"strconv"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// dictKey returns a Go map key that is equal exactly when Python treats two values as
// the same dict key or set member (1, 1.0 and True are one key; "1" is another).
func dictKey(value any) string {
	switch v := value.(type) {
	case nil:
		return "z"
	case string:
		return "s" + v
	case bool:
		if v {
			return "n1"
		}
		return "n0"
	case *big.Int:
		return "n" + v.String()
	case float64:
		if !math.IsInf(v, 0) && !math.IsNaN(v) && v == math.Trunc(v) {
			whole, _ := new(big.Float).SetFloat64(v).Int(nil)
			return "n" + whole.String()
		}
		return "f" + pyv.Repr(v)
	}
	if number, ok := pyv.Int64(value); ok {
		return "n" + strconv.FormatInt(number, 10)
	}
	return "r" + pyv.Repr(value)
}

// tupleKey is dictKey for a tuple of values.
func tupleKey(parts ...any) string {
	key := ""
	for _, part := range parts {
		key += dictKey(part) + "\x00"
	}
	return key
}

// equalsAny is Python's `value in (a, b, ...)` for a tuple of constants.
func equalsAny(value any, options ...any) bool {
	for _, option := range options {
		if pyv.Equal(value, option) {
			return true
		}
	}
	return false
}

// strOr is object.get(key, fallback) used as text: str() of a non-string value.
func strOr(object any, key string, fallback string) string {
	return pyv.PyStr(pyv.GetOr(pyv.Obj(object), key, fallback))
}

// objects returns object.get(key, []) as objects.
func objects(object *pyjson.Object, key string) []*pyjson.Object {
	return pyv.Objects(pyv.List(pyv.GetOr(object, key, []any{})))
}
