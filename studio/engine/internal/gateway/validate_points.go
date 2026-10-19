package gateway

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// AllowedKinds is allowed_kinds(value_kind), sorted: the gateway kinds that carry an
// IEC 61850 value kind without changing its meaning.
func AllowedKinds(valueKind string) []string {
	switch {
	case valueKind == "BOOLEAN":
		return []string{"single"}
	case valueKind == "Dbpos":
		return []string{"double"}
	case strings.HasPrefix(valueKind, "FLOAT"):
		return []string{"float"}
	case strings.HasPrefix(valueKind, "INT") || valueKind == "Enum":
		return []string{"float", "integer"}
	}
	return []string{}
}

// allowedKindsOf is allowed_kinds for any value (AttributeError when it is not a str).
func allowedKindsOf(value any) ([]string, error) {
	if value == "BOOLEAN" || value == "Dbpos" {
		return AllowedKinds(value.(string)), nil
	}
	text, ok := value.(string)
	if !ok {
		return nil, noAttribute(value, "startswith")
	}
	return AllowedKinds(text), nil
}

// pointLookup holds ieds_by_id and the per-IED signal index built on first use.
type pointLookup struct {
	byID    map[string]*pyjson.Object
	signals map[string]map[string]*pyjson.Object
}

func newLookup(ieds []*pyjson.Object) (*pointLookup, error) {
	lookup := &pointLookup{byID: map[string]*pyjson.Object{}, signals: map[string]map[string]*pyjson.Object{}}
	for _, ied := range ieds {
		id, err := subscript(ied, "id")
		if err != nil {
			return nil, err
		}
		if err := hashable(id); err != nil {
			return nil, err
		}
		if text, ok := id.(string); ok { // cleaned points carry str IED ids only
			lookup.byID[text] = ied
		}
	}
	return lookup, nil
}

// target is _target(point, ...): the point's IED (nil for None) and the reason it is unusable.
func (lookup *pointLookup) target(kind, iedID, ref string) (*pyjson.Object, string, error) {
	ied := lookup.byID[iedID]
	if ied == nil {
		return nil, "its IED is not in the project", nil
	}
	if IsCommandKind(kind) {
		reason, err := controlReason(ied, ref)
		return ied, reason, err
	}
	if _, built := lookup.signals[iedID]; !built {
		index, err := refIndex(ied, "signals", nil)
		if err != nil {
			return nil, "", err
		}
		lookup.signals[iedID] = index
	}
	signal := lookup.signals[iedID][ref]
	if signal == nil {
		name, err := subscript(ied, "name")
		return ied, pytext.Str(name) + " has no attribute " + ref, err
	}
	valueKind, err := subscript(signal, "kind")
	if err != nil {
		return nil, "", err
	}
	allowed, err := allowedKindsOf(valueKind)
	if err != nil {
		return nil, "", err
	}
	if !contains(allowed, kind) {
		return ied, ref + " is a " + pytext.Str(valueKind) + " value and cannot be sent as " + kind, nil
	}
	return ied, "", nil
}

// controlReason is why a command point cannot use the IED ("" when it can).
func controlReason(ied *pyjson.Object, ref string) (string, error) {
	controls, err := refIndex(ied, "controls", []any{})
	if err != nil || controls[ref] != nil {
		return "", err
	}
	name, err := subscript(ied, "name")
	return pytext.Str(name) + " has no controllable object " + ref, err
}

// refIndex is {item["ref"]: item for item in ied[key]} (ied.get(key, fallback) when fallback
// is not nil); later items win and only str references can match a point reference.
func refIndex(ied *pyjson.Object, key string, fallback any) (map[string]*pyjson.Object, error) {
	var list any
	var err error
	if fallback != nil {
		list = getOr(ied, key, fallback)
	} else if list, err = subscript(ied, key); err != nil {
		return nil, err
	}
	items, err := iterate(list)
	if err != nil {
		return nil, err
	}
	index := map[string]*pyjson.Object{}
	for _, item := range items {
		ref, err := subscript(item, "ref")
		if err != nil {
			return nil, err
		}
		if err := hashable(ref); err != nil {
			return nil, err
		}
		if text, ok := ref.(string); ok {
			index[text] = item.(*pyjson.Object)
		}
	}
	return index, nil
}

// scaleProblem is the scale test of validate_points; OverflowError for huge ints.
func scaleProblem(scale any) (bool, error) {
	switch v := scale.(type) {
	case float64:
		return math.IsNaN(v) || math.IsInf(v, 0) || v == 0, nil
	case int, int64:
		number, _ := pytext.Int64(v)
		return number == 0, nil
	case *big.Int:
		_, err := bigFloat(v)
		return false, err
	}
	return true, nil // bool, None, str, list, dict
}

func itoa(number int) string { return strconv.Itoa(number) }
