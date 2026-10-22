package addpoints

import (
	"strconv"
	"strings"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// SimulatedNames are what the IED column says for a made-up (own) point.
var SimulatedNames = []string{"", "simulated", "simulated (made up)", "made up", "own point", "own"}

// CheckRows reads pasted or Excel rows into new points and checks every one before anything is
// added: the type must be one of the protocol, the IED a project IED (or empty / "Simulated"
// for a made-up point), the point one of that IED's signals or commands that fits the type,
// the address free and in range. Typed addresses are taken first; empty ones get the next
// free address. numbers are the rows' line numbers in the source.
func CheckRows(protocol string, rows []Row, numbers []int, index *Index, ieds []*pyjson.Object, current []any) []Result {
	addresses := newAddresser(protocol, current)
	taken := TakenIDs(current)
	own := nextOwnNumber(current)
	results := make([]Result, len(rows))
	kinds := make([]string, len(rows))
	for at, row := range rows { // pass 1: what each row is, and its typed address
		cells := row
		results[at] = Result{Number: numbers[at], Cells: &cells}
		kind, problem := readKind(protocol, row.Type)
		kinds[at] = kind
		var point *pyjson.Object
		if problem == "" {
			point, problem = rowPoint(protocol, kind, row, index, taken)
		}
		if problem == "" && row.Address != "" {
			address, err := strconv.ParseInt(strings.ReplaceAll(row.Address, " ", ""), 10, 64)
			if err != nil {
				problem = "the address must be a whole number"
			} else if problem = addresses.claim(kind, address); problem == "" {
				point.Set("address", address)
			}
		}
		if point == nil {
			point = pyjson.NewObject().Set("kind", kind).Set("description", row.Name)
		}
		results[at].Point, results[at].Problem = point, problem
	}
	for at := range rows { // pass 2: the next free address for rows without one
		result := &results[at]
		if result.Problem != "" {
			continue
		}
		if _, has := result.Point.Get("address"); !has {
			if next, ok := addresses.next(kinds[at]); ok {
				result.Point.Set("address", next)
			} else {
				result.Problem = "no free address is left for this type"
			}
		}
		if gateway.IsOwn(result.Point) {
			result.Point.Set("id", gateway.OwnID(own))
			own++
		}
	}
	for at := range results {
		if results[at].Problem != "" {
			results[at].Problem = "Row " + strconv.Itoa(results[at].Number) + ": " + results[at].Problem
		}
	}
	checkWithEngine(protocol, results, ieds)
	return results
}

func readKind(protocol, typed string) (string, string) {
	if strings.TrimSpace(typed) == "" {
		return "", "choose a type"
	}
	kind, ok := ResolveType(protocol, typed)
	if !ok {
		names := []string{}
		for _, entry := range Types(protocol) {
			names = append(names, entry.Name)
		}
		return "", "\"" + typed + "\" is not a type here; use one of: " + strings.Join(names, "; ") // names hold commas
	}
	return kind, ""
}

// rowPoint is the new point a row describes (without its address), or why there is none.
func rowPoint(protocol, kind string, row Row, index *Index, taken map[string]bool) (*pyjson.Object, string) {
	scale := 1.0
	if row.Scale != "" {
		value, err := strconv.ParseFloat(strings.ReplaceAll(row.Scale, ",", "."), 64)
		if err != nil || value == 0 {
			return nil, "the scale must be a number other than 0"
		}
		scale = value
	}
	if contains(SimulatedNames, strings.ToLower(strings.TrimSpace(row.IED))) {
		if row.Point != "" {
			return nil, "a made-up point has no IED path; write the IED name, or clear the Point cell"
		}
		point := pyjson.NewObject().Set("kind", kind).Set("source", gateway.OwnSource).Set("ied", "").Set("ref", "").
			Set("scale", scale).Set("description", head(row.Name, 120)).Set("ied_name", gateway.OwnName).
			Set("ied_role", "own").Set("monitor_only", false)
		if value := gateway.DefaultOwnValue(kind); value != nil {
			point.Set("value", value)
		}
		return point, ""
	}
	ied := index.Find(row.IED)
	if ied == nil {
		return nil, "this project has no IED called \"" + row.IED + "\""
	}
	if row.Point == "" {
		return nil, "write which " + ied.Name + " signal or command this point is (the Point column)"
	}
	ref := ied.FullRef(row.Point, gateway.IsCommandKind(kind))
	if problem := fitProblem(ied, kind, ref); problem != "" {
		return nil, problem
	}
	id := gateway.PointID(ied.ID, ref, kind)
	if taken[id] {
		return nil, "this " + typeName(protocol, kind) + " is already in the list"
	}
	taken[id] = true
	point := iedPoint(ied, kind, ref, nameOf(ied, ref, row.Name), 0, scale)
	point.Delete("address") // set in pass 1 (typed) or pass 2 (next free)
	return point, ""
}

// nextOwnNumber is the first own point number above every "own:<n>" of the list.
func nextOwnNumber(points []any) int {
	next := 1
	for _, raw := range points {
		if point, ok := raw.(*pyjson.Object); ok && strings.HasPrefix(text(point, "id"), "own:") {
			if number, err := strconv.Atoi(strings.TrimPrefix(text(point, "id"), "own:")); err == nil && number >= next {
				next = number + 1
			}
		}
	}
	return next
}
