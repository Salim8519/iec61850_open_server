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
	for index, value := range list {
		number := index + 1
		label := "Point " + itoa(number)
		raw, isDict := value.(*pyjson.Object)
		if !isDict || !contains(Kinds, get(raw, "kind")) {
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
		point := pyjson.NewObject().Set("kind", kind).
			Set("ied", pytext.Str(getOr(raw, "ied", ""))).Set("ref", pytext.Str(getOr(raw, "ref", ""))).
			Set("address", at).Set("scale", factor).
			Set("description", pytext.Head(pytext.Str(getOr(raw, "description", "")), 120))
		iedID, ref := get(point, "ied").(string), get(point, "ref").(string)
		ied, reason, err := lookup.target(kind, iedID, ref)
		if err != nil {
			return nil, nil, err
		}
		name := "?"
		if ied != nil {
			value, err := subscript(ied, "name")
			if err != nil {
				return nil, nil, err
			}
			name = pytext.Str(value)
		}
		label += " (" + name + " " + ref + ")"
		if reason != "" {
			problems = append(problems, label+": "+reason)
			continue
		}
		taken := cells(protocol, kind, at)
		if clash := firstClash(used, taken); clash != 0 {
			problems = append(problems, label+": address "+strconv.FormatInt(at, 10)+
				" is already used by point "+itoa(clash))
			continue
		}
		for _, cell := range taken {
			used[cell] = number
		}
		point.Set("id", PointID(iedID, ref, kind))
		clean = append(clean, point)
	}
	return clean, problems, nil
}

func firstClash(used map[Cell]int, taken []Cell) int {
	for _, cell := range taken {
		if number, found := used[cell]; found {
			return number
		}
	}
	return 0
}
