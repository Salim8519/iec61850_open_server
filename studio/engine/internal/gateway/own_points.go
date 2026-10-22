package gateway

import (
	"math"
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Own points (lane G, Go-only; the frozen Python engine has none): a point the user makes
// up. It has no IED; ScadaBench serves the value stored with the point ("value") with good
// quality while the gateway runs. A command on an own command point is a simulation: its
// value is stored and the own status point it drives ("target") follows.

// OwnSource is the "source" of an own point.
const OwnSource = "manual"

const ownPrefix = "own:"

// OwnName is what the IED column says for an own point.
const OwnName = "Own point"

// IsOwn reports whether a point is an own point.
func IsOwn(point *pyjson.Object) bool {
	return point != nil && getOr(point, "source", nil) == OwnSource
}

// OwnID is the id of own point number n: "own:<n>". IED point ids always end with
// ":<kind>", so the two never clash.
func OwnID(number int) string { return ownPrefix + strconv.Itoa(number) }

// ownNumber is n of a well-formed "own:<n>" (n from 1, no leading zero).
func ownNumber(id any) (int, bool) {
	text, ok := id.(string)
	if !ok || !strings.HasPrefix(text, ownPrefix) {
		return 0, false
	}
	digits := text[len(ownPrefix):]
	if digits == "" || digits[0] == '0' || len(digits) > 9 {
		return 0, false
	}
	number, err := strconv.Atoi(digits)
	return number, err == nil
}

// DefaultOwnValue is the value a new own point starts with (a command has none yet).
func DefaultOwnValue(kind string) any {
	switch kind {
	case "double":
		return int64(1)
	case "float":
		return 0.0
	case "single", "integer", KindStepPosition:
		return int64(0)
	}
	return nil
}

// OwnValue checks a value for an own point of kind: the stored value, or a plain reason.
func OwnValue(kind string, raw any) (any, string) {
	flag, isBool := raw.(bool)
	switch kind {
	case "single", "command_single":
		if isBool {
			return boolInt(flag, 0, 1), ""
		}
		if pytext.IntIn(raw, 0, 1) {
			number, _ := pytext.Int64(raw)
			return number, ""
		}
		return nil, "the value must be OFF (0) or ON (1)"
	case "double", "command_double":
		if isBool {
			return boolInt(flag, 1, 2), ""
		}
		if pytext.IntIn(raw, 0, 3) {
			number, _ := pytext.Int64(raw)
			return number, ""
		}
		return nil, "the value must be 0 (intermediate), 1 (OFF / open), 2 (ON / closed) or 3 (bad state)"
	case "float":
		number, ok := pytext.Float(raw)
		if isBool || !ok || math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, "the value must be a number"
		}
		return number, ""
	case "integer":
		if number, ok := pytext.Float(raw); ok && !isBool && number == math.Trunc(number) &&
			number >= math.MinInt32 && number <= math.MaxInt32 {
			return int64(number), ""
		}
		return nil, "the value must be a whole number from -2147483648 to 2147483647"
	}
	return typedOwnValue(kind, raw)
}

func boolInt(flag bool, off, on int64) int64 {
	if flag {
		return on
	}
	return off
}

// ownPoint is the clean own point of a raw one whose kind, address and scale are checked.
func ownPoint(raw *pyjson.Object, kind string, at int64, factor float64) (*pyjson.Object, string) {
	point := pyjson.NewObject().Set("kind", kind).Set("source", OwnSource).Set("ied", "").Set("ref", "").
		Set("address", at).Set("scale", factor).
		Set("description", pytext.Head(pytext.Str(getOr(raw, "description", "")), 120))
	value := getOr(raw, "value", nil)
	if value == nil {
		value = DefaultOwnValue(kind)
	}
	if value != nil {
		checked, reason := OwnValue(kind, value)
		if reason != "" {
			return nil, reason
		}
		point.Set("value", checked)
	}
	if target, _ := getOr(raw, "target", nil).(string); target != "" && IsCommandKind(kind) {
		point.Set("target", target)
	}
	return point, ""
}

// ownLabel names an own point in a problem sentence.
func ownLabel(raw *pyjson.Object) string {
	if name := pytext.Head(pytext.Str(getOr(raw, "description", "")), 120); name != "" {
		return " (own point " + name + ")"
	}
	return " (own point)"
}

// ownIDs hands out own point ids: a well-formed id keeps its number the first time it is
// seen; a missing or repeated one gets the next number above every given one.
type ownIDs struct {
	used map[string]bool
	next int
}

func newOwnIDs(list []any) *ownIDs {
	ids := &ownIDs{used: map[string]bool{}, next: 1}
	for _, value := range list {
		if raw, isDict := value.(*pyjson.Object); isDict && IsOwn(raw) {
			if number, ok := ownNumber(getOr(raw, "id", nil)); ok && number >= ids.next {
				ids.next = number + 1
			}
		}
	}
	return ids
}

func (ids *ownIDs) assign(raw *pyjson.Object) string {
	if given, _ := getOr(raw, "id", nil).(string); given != "" && !ids.used[given] {
		if _, ok := ownNumber(given); ok {
			ids.used[given] = true
			return given
		}
	}
	id := OwnID(ids.next)
	ids.next++
	ids.used[id] = true
	return id
}

// ownEntry is a clean own point with its number and label for later problems.
type ownEntry struct {
	label string
	point *pyjson.Object
}

// checkTargets drops a command's target that is not an own status point of the list.
func checkTargets(owned []ownEntry) []string {
	kinds := map[string]string{}
	for _, entry := range owned {
		kinds[get(entry.point, "id").(string)] = get(entry.point, "kind").(string)
	}
	problems := []string{}
	for _, entry := range owned {
		target, has := getOr(entry.point, "target", "").(string)
		if !has || target == "" {
			continue
		}
		command := get(entry.point, "kind").(string)
		if reason := targetReason(command, kinds[target], target); reason != "" {
			entry.point.Delete("target")
			problems = append(problems, entry.label+": "+reason)
		}
	}
	return problems
}
