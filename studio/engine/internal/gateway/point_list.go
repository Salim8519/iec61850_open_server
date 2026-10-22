package gateway

import (
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// ValidatePoints is validate_points(protocol, points, ieds): the clean points and the
// problems, one sentence per refused point that names the point and the fix.
func ValidatePoints(protocol string, points any, ieds []*pyjson.Object) ([]any, []string, error) {
	list, ok := points.([]any)
	if !ok {
		return nil, nil, valueError("Send the point list as a list")
	}
	if len(list) > MaxPoints {
		return nil, nil, valueError("A gateway holds at most " + strconv.Itoa(MaxPoints) + " points")
	}
	lookup, err := newLookup(ieds)
	if err != nil {
		return nil, nil, err
	}
	bounds, ok := AddressRange[protocol]
	if !ok {
		return nil, nil, keyError(protocol)
	}
	low, high := bounds[0], bounds[1]
	clean, problems, used := []any{}, []string{}, map[Cell]int{}
	ids, owned := newOwnIDs(list), []ownEntry{}
	for index, value := range list {
		number := index + 1
		label := "Point " + itoa(number)
		raw, isDict := value.(*pyjson.Object)
		if !isDict || !KnownKind(get(raw, "kind")) {
			problems = append(problems, label+": choose a type ("+strings.Join(Kinds, ", ")+")")
			continue
		}
		kind := get(raw, "kind").(string)
		address, scale := get(raw, "address"), getOr(raw, "scale", 1.0)
		if !pytext.IntIn(address, low, high) {
			problems = append(problems, label+": the address must be a whole number from "+
				strconv.FormatInt(low, 10)+" to "+strconv.FormatInt(high, 10))
			continue
		}
		bad, err := scaleProblem(scale)
		if err != nil {
			return nil, nil, err
		}
		if bad {
			problems = append(problems, label+": the scale must be a number other than 0")
			continue
		}
		at, _ := pytext.Int64(address)
		factor, _ := pyFloat(scale)
		var point *pyjson.Object
		var reason string
		own := IsOwn(raw)
		if own { // Go-only: an own point has no IED to look up
			point, reason = ownPoint(raw, kind, at, factor)
			label += ownLabel(raw)
		} else {
			point = pyjson.NewObject().Set("kind", kind).
				Set("ied", pytext.Str(getOr(raw, "ied", ""))).Set("ref", pytext.Str(getOr(raw, "ref", ""))).
				Set("address", at).Set("scale", factor).
				Set("description", pytext.Head(pytext.Str(getOr(raw, "description", "")), 120))
			var named string
			if named, reason, err = iedLabel(lookup, point, kind); err != nil {
				return nil, nil, err
			}
			label += named
		}
		if reason == "" && IsTypedKind(kind) { // Go only (ADR-0107)
			reason = typedReason(protocol, kind, raw, point)
		}
		if reason != "" {
			problems = append(problems, label+": "+reason)
			continue
		}
		format, _ := getOr(point, "format", "").(string)
		taken := cellsOf(protocol, kind, format, at)
		if clash := firstClash(used, taken); clash != 0 {
			problems = append(problems, label+": address "+strconv.FormatInt(at, 10)+
				" is already used by point "+itoa(clash))
			continue
		}
		for _, cell := range taken {
			used[cell] = number
		}
		if own {
			point.Set("id", ids.assign(raw))
			owned = append(owned, ownEntry{label: label, point: point})
		} else {
			point.Set("id", PointID(get(point, "ied").(string), get(point, "ref").(string), kind))
		}
		clean = append(clean, point)
	}
	return clean, append(problems, checkTargets(owned)...), nil
}

// iedLabel looks up the point's IED: " (<IED name> <ref>)" and why the point is unusable.
func iedLabel(lookup *pointLookup, point *pyjson.Object, kind string) (string, string, error) {
	iedID, ref := get(point, "ied").(string), get(point, "ref").(string)
	ied, reason, err := lookup.target(kind, iedID, ref)
	if err != nil {
		return "", "", err
	}
	name := "?"
	if ied != nil {
		value, err := subscript(ied, "name")
		if err != nil {
			return "", "", err
		}
		name = pytext.Str(value)
	}
	return " (" + name + " " + ref + ")", reason, nil
}

func firstClash(used map[Cell]int, taken []Cell) int {
	for _, cell := range taken {
		if number, found := used[cell]; found {
			return number
		}
	}
	return 0
}
