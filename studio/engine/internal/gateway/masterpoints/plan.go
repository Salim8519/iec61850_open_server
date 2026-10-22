package masterpoints

import (
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
)

// Result is one planned row: the point as it would be added and, when it cannot be added, the
// plain reason. Nothing is added while any row has a problem (as the gateway's Add points).
type Result struct {
	Number  int // 1-based row of the request, the pasted text or the sheet
	Point   *pyjson.Object
	Problem string
	Cells   *pyjson.Object // the cells a pasted or Excel row was read from
}

// OK reports whether there are rows and every one can be added.
func OK(results []Result) bool {
	for _, result := range results {
		if result.Problem != "" {
			return false
		}
	}
	return len(results) > 0
}

// PlanJSON is the answer of every planning route: {rows, points, ok}; points holds the new
// points only when every row is right.
func PlanJSON(results []Result) *pyjson.Object {
	rows, points := []any{}, []any{}
	for _, result := range results {
		row := pyjson.NewObject().Set("number", int64(result.Number)).Set("point", result.Point).
			Set("problem", result.Problem)
		if result.Cells != nil {
			row.Set("cells", result.Cells)
		}
		rows = append(rows, row)
		if result.Point != nil {
			points = append(points, result.Point)
		}
	}
	ok := OK(results)
	if !ok {
		points = []any{}
	}
	return pyjson.NewObject().Set("rows", rows).Set("points", points).Set("ok", ok)
}

// planner checks new points against the current list and each other, and numbers their ids
// after the list's ids, so a plan can be appended to the list as it is.
type planner struct {
	protocol string
	use      Use
	names    []string // by 1-based point number: the current list, then the planned points
	next     int
}

func newPlanner(protocol string, current []any) *planner {
	p := &planner{protocol: protocol, use: UseOf(protocol, current), next: 1}
	for _, raw := range current {
		point, _ := raw.(*pyjson.Object)
		p.names = append(p.names, text(point, "name"))
		if number, err := strconv.Atoi(strings.TrimPrefix(text(point, "id"), "m")); err == nil && number >= p.next {
			p.next = number + 1
		}
	}
	return p
}

// add checks one raw point and claims its cells; it returns the stored point or the problem.
func (p *planner) add(raw *pyjson.Object) (*pyjson.Object, string) {
	if len(p.names) >= MaxPoints {
		return nil, "the list already has " + strconv.Itoa(MaxPoints) + " points"
	}
	point, problem := CheckPoint(p.protocol, raw)
	if problem != "" {
		return point, problem
	}
	cells := Cells(p.protocol, point)
	if other := p.use.Taken(cells); other > 0 {
		return point, AddressWord[p.protocol] + " " + strconv.FormatInt(Address(point), 10) + " is already used" +
			p.byWhom(other)
	}
	p.names = append(p.names, text(point, "name"))
	p.use.claim(cells, len(p.names))
	point.Set("id", "m"+strconv.Itoa(p.next))
	p.next++
	return point, ""
}

func (p *planner) byWhom(number int) string {
	if number > 0 && number <= len(p.names) && p.names[number-1] != "" {
		return " by \"" + p.names[number-1] + "\""
	}
	return " by another point"
}

// rowProblem prefixes a problem with its row number.
func rowProblem(number int, problem string) string {
	if problem == "" {
		return ""
	}
	return "Row " + strconv.Itoa(number) + ": " + problem
}
