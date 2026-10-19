package gateway

import (
	"fmt"
	"sort"

	"studio-engine/internal/pyjson"
)

type candidate struct {
	kind    string
	ied     any
	ref     any
	generic bool
	rank    int
}

// Propose is propose(ieds, protocol): the smart selection of every IED that is not
// switched off, addressed by one simple plan per protocol. Malformed IED models give the
// Python crash (KeyError/AttributeError/TypeError) as a plain error.
func Propose(ieds []*pyjson.Object, protocol string) ([]any, []string, error) {
	if !contains(Protocols, protocol) {
		return nil, nil, valueError("Choose IEC 104, Modbus TCP or DNP3")
	}
	candidates, err := collect(ieds)
	if err != nil {
		return nil, nil, err
	}
	for index := range candidates { // Python computes every sort key before sorting
		item := &candidates[index]
		if item.generic, item.rank, err = rankOf(item.kind, item.ref); err != nil {
			return nil, nil, err
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.generic != b.generic {
			return !a.generic
		}
		return a.rank < b.rank
	})
	notes := []string{}
	if len(candidates) > MaxPoints {
		notes = append(notes, fmt.Sprintf("The IEDs offer %d points; the first %d were proposed. "+
			"Remove what SCADA does not need, or import your own point list as CSV.", len(candidates), MaxPoints))
		candidates = candidates[:MaxPoints]
	}
	points := address(candidates, protocol)
	if len(points) == 0 {
		notes = append(notes, "No position, status, measurement or command was found. Import IEDs first, "+
			"or add points from a CSV point list.")
	}
	return points, notes, nil
}

// collect lists the candidates in IED order: useful signals, then controllable objects.
func collect(ieds []*pyjson.Object) ([]candidate, error) {
	candidates := []candidate{}
	for _, ied := range ieds {
		if get(ied, "role") == "off" {
			continue
		}
		refs, err := datasetRefs(ied)
		if err != nil {
			return nil, err
		}
		signals, err := subscript(ied, "signals")
		if err == nil {
			signals, err = iterate(signals)
		}
		if err != nil {
			return nil, err
		}
		for _, value := range signals.([]any) {
			signal, ok := value.(*pyjson.Object)
			if !ok {
				_, err := subscript(value, "ref")
				return nil, err
			}
			kind, err := smartKind(signal)
			if err != nil {
				return nil, err
			}
			if kind != "" && Engineered(signal, refs) {
				id, err := subscript(ied, "id")
				if err != nil {
					return nil, err
				}
				candidates = append(candidates, candidate{kind: kind, ied: id, ref: get(signal, "ref")})
			}
		}
		controls, err := iterate(getOr(ied, "controls", []any{}))
		if err != nil {
			return nil, err
		}
		for _, value := range controls {
			cdc, err := getAny(value, "cdc", nil)
			if err != nil {
				return nil, err
			}
			kind := "command_single"
			if cdc == "DPC" {
				kind = "command_double"
			}
			id, err := subscript(ied, "id")
			if err != nil {
				return nil, err
			}
			ref, err := subscript(value, "ref")
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, candidate{kind: kind, ied: id, ref: ref})
		}
	}
	return candidates, nil
}

// address gives every candidate the next free address of its block, in KINDS order.
func address(candidates []candidate, protocol string) []any {
	order := map[string]int{}
	for index, kind := range Kinds {
		order[kind] = index
	}
	sort.SliceStable(candidates, func(i, j int) bool { return order[candidates[i].kind] < order[candidates[j].kind] })
	used, points, cursor := map[Cell]bool{}, []any{}, map[string]int64{}
	for block, first := range startAddress[protocol] {
		cursor[block] = first
	}
	for _, item := range candidates {
		block := item.kind
		if IsCommandKind(block) {
			block = "command"
		}
		at := cursor[block]
		for anyUsed(used, cells(protocol, item.kind, at)) {
			at++
		}
		taken := cells(protocol, item.kind, at)
		for _, cell := range taken {
			used[cell] = true
		}
		cursor[block] = taken[len(taken)-1].Address + 1 // never scans from the start again
		point := pyjson.NewObject().Set("id", pointIDOf(item.ied, item.ref, item.kind)).
			Set("kind", item.kind).Set("ied", item.ied).Set("ref", item.ref).
			Set("address", at).Set("scale", 1.0).Set("description", "")
		points = append(points, point)
	}
	return points
}

func anyUsed(used map[Cell]bool, list []Cell) bool {
	for _, cell := range list {
		if used[cell] {
			return true
		}
	}
	return false
}
