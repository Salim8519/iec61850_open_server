package gateway

import (
	"strconv"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// NewOwnPoint is a new own point of kind for the point list (lane G): the next own id and
// the first free address of the kind's block, counted the way Propose numbers points.
func NewOwnPoint(protocol, kind string, points any) (*pyjson.Object, error) {
	if !contains(Protocols, protocol) {
		return nil, valueError("Choose IEC 104, Modbus TCP or DNP3")
	}
	if !contains(Kinds, kind) {
		return nil, valueError("Choose a point type")
	}
	list, _ := points.([]any)
	block := kind
	if IsCommandKind(kind) {
		block = "command"
	}
	at, free := freeAddress(protocol, kind, startAddress[protocol][block], usedCells(protocol, list))
	if !free {
		return nil, valueError("No free address is left for this type of point")
	}
	point := pyjson.NewObject().Set("id", OwnID(newOwnIDs(list).next)).Set("kind", kind).Set("source", OwnSource).
		Set("ied", "").Set("ref", "").Set("address", at).Set("scale", 1.0).Set("description", "")
	if value := DefaultOwnValue(kind); value != nil {
		point.Set("value", value)
	}
	return point, nil
}

// KeepOwn adds the own points of keep after a proposal, so Propose from IEDs never removes
// them. An own point whose address a proposed point now uses moves to the next free one.
func KeepOwn(protocol string, proposal []any, keep any) ([]any, []string) {
	list, _ := keep.([]any)
	used := usedCells(protocol, proposal)
	result, notes := append([]any{}, proposal...), []string{}
	for _, value := range list {
		raw, isDict := value.(*pyjson.Object)
		if !isDict || !IsOwn(raw) {
			continue
		}
		point := copyObject(raw)
		kind, _ := get(point, "kind").(string)
		if at, isInt := pytext.Int64(get(point, "address")); isInt && pytext.IsInt(get(point, "address")) &&
			contains(Kinds, kind) {
			if anyUsed(used, cells(protocol, kind, at)) {
				if moved, free := freeAddress(protocol, kind, at, used); free {
					notes = append(notes, ownName(point)+" moved from address "+strconv.FormatInt(at, 10)+" to "+
						strconv.FormatInt(moved, 10)+" because a proposed point uses "+strconv.FormatInt(at, 10)+".")
					point.Set("address", moved)
					at = moved
				}
			}
			for _, cell := range cells(protocol, kind, at) {
				used[cell] = true
			}
		}
		result = append(result, point)
	}
	return result, notes
}

func ownName(point *pyjson.Object) string {
	if name := pytext.Str(getOr(point, "description", "")); name != "" {
		return "Own point \"" + name + "\""
	}
	return "An own point"
}

// usedCells are the cells the well-formed points of a list occupy.
func usedCells(protocol string, list []any) map[Cell]bool {
	used := map[Cell]bool{}
	for _, value := range list {
		point, isDict := value.(*pyjson.Object)
		if !isDict {
			continue
		}
		kind, _ := get(point, "kind").(string)
		at, isInt := pytext.Int64(get(point, "address"))
		if !contains(Kinds, kind) || !isInt || !pytext.IsInt(get(point, "address")) {
			continue
		}
		for _, cell := range cells(protocol, kind, at) {
			used[cell] = true
		}
	}
	return used
}

// freeAddress is the first address from "from" whose cells are free and in range.
func freeAddress(protocol, kind string, from int64, used map[Cell]bool) (int64, bool) {
	bounds := AddressRange[protocol]
	if from < bounds[0] {
		from = bounds[0]
	}
	last := bounds[1] - width(protocol, kind) + 1
	for at := from; at <= last; at++ {
		if !anyUsed(used, cells(protocol, kind, at)) {
			return at, true
		}
	}
	return 0, false
}
