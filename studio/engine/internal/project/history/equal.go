package history

import (
	"reflect"

	"studio-engine/internal/pyjson"
)

// Identical reports whether a and b are the same value: the same object or list (project
// trees are copy-on-write, so an unchanged part keeps its identity) or equal plain values.
func Identical(a, b any) bool {
	switch x := a.(type) {
	case *pyjson.Object:
		y, ok := b.(*pyjson.Object)
		return ok && x == y
	case []any:
		y, ok := b.([]any)
		return ok && len(x) == len(y) && (len(x) == 0 || &x[0] == &y[0])
	}
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	kind := reflect.TypeOf(a)
	return kind == reflect.TypeOf(b) && kind.Comparable() && a == b
}

// Equal compares a and b down to depth levels below identity: objects key by key, lists item
// by item. ignore names object fields left out of the comparison; it applies to a itself
// when a is an object, and to the items when a is a list (an IED's field in the IED list).
// Edits copy the objects on their path, so a few levels tell a real change from a copy.
func Equal(a, b any, depth int, ignore func(field string) bool) bool {
	if Identical(a, b) {
		return true
	}
	if depth == 0 {
		return false
	}
	switch x := a.(type) {
	case *pyjson.Object:
		y, ok := b.(*pyjson.Object)
		return ok && x != nil && y != nil && sameFields(x, y, depth, ignore) && sameFields(y, x, depth, ignore)
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for index := range x {
			if !Equal(x[index], y[index], depth-1, ignore) {
				return false
			}
		}
		return true
	}
	return false
}

// sameFields says every compared field of x is in y with an equal value.
func sameFields(x, y *pyjson.Object, depth int, ignore func(string) bool) bool {
	for _, key := range x.Keys() {
		if ignore != nil && ignore(key) {
			continue
		}
		left, _ := x.Get(key)
		right, ok := y.Get(key)
		if !ok || !Equal(left, right, depth-1, nil) {
			return false
		}
	}
	return true
}
