package scl

import (
	"math/big"

	"studio-engine/internal/pyjson"
)

// sameSignals is the exporter's `actual != wanted` check: the same references, each with
// the same kind, fc and (Python ==) value.
func sameSignals(restored []any, original map[string]*pyjson.Object, expected map[string]any) bool {
	actual := map[string]*pyjson.Object{}
	for _, item := range restored {
		signal := item.(*pyjson.Object)
		actual[get(signal, "ref").(string)] = signal
	}
	if len(actual) != len(original) {
		return false
	}
	for ref, signal := range original {
		got := actual[ref]
		if got == nil || get(got, "kind") != get(signal, "kind") || get(got, "fc") != get(signal, "fc") ||
			!pyEqual(get(got, "value"), expected[ref]) {
			return false
		}
	}
	return true
}

// pyEqual is Python == over decoded JSON-like values: numbers compare by value across
// bool/int/float, dicts ignore key order, lists compare in order.
func pyEqual(a, b any) bool {
	if x, ok := number(a); ok {
		y, ok := number(b)
		return ok && x.Cmp(y) == 0
	}
	switch av := a.(type) {
	case nil:
		return b == nil
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !pyEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case *pyjson.Object:
		bv, ok := b.(*pyjson.Object)
		if !ok || av.Len() != bv.Len() {
			return false
		}
		for _, key := range av.Keys() {
			x, _ := av.Get(key)
			y, found := bv.Get(key)
			if !found || !pyEqual(x, y) {
				return false
			}
		}
		return true
	}
	return false
}

// number converts bool, int and float values to an exact rational for comparison.
func number(v any) (*big.Rat, bool) {
	switch n := v.(type) {
	case bool:
		if n {
			return big.NewRat(1, 1), true
		}
		return new(big.Rat), true
	case int:
		return big.NewRat(int64(n), 1), true
	case int64:
		return big.NewRat(n, 1), true
	case *big.Int:
		return new(big.Rat).SetInt(n), true
	case float64:
		r := new(big.Rat).SetFloat64(n) // exact, as Python compares int and float
		return r, r != nil
	}
	return nil, false
}
