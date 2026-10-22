package addpoints

import (
	"fmt"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// Result is one row of a planned addition: the point as it would be added and, when it
// cannot be added, the plain reason. Nothing is added while any row has a problem.
type Result struct {
	Number  int // 1-based row of the request, the pasted text or the sheet
	Point   *pyjson.Object
	Problem string
	// StatusFor is set on a status point added with its command: the command's point id.
	StatusFor string
	// Cells are the values the row was read from (pasted text and Excel rows only).
	Cells *Row
}

// OK reports whether every row can be added.
func OK(results []Result) bool {
	for _, result := range results {
		if result.Problem != "" {
			return false
		}
	}
	return len(results) > 0
}

// Points are the points of the rows, in order.
func Points(results []Result) []any {
	points := make([]any, 0, len(results))
	for _, result := range results {
		points = append(points, result.Point)
	}
	return points
}

// JSON is the answer shape of the rows: {number, point, problem[, status_for][, cells]}.
func JSON(results []Result) []any {
	rows := make([]any, 0, len(results))
	for _, result := range results {
		row := pyjson.NewObject().Set("number", int64(result.Number)).Set("point", result.Point).
			Set("problem", result.Problem)
		if result.StatusFor != "" {
			row.Set("status_for", result.StatusFor)
		}
		if result.Cells != nil {
			row.Set("cells", result.Cells.JSON())
		}
		rows = append(rows, row)
	}
	return rows
}

// iedPoint is a new point of an IED signal or controllable object.
func iedPoint(ied *IED, kind, ref, name string, address int64, scale float64) *pyjson.Object {
	return pyjson.NewObject().Set("id", gateway.PointID(ied.ID, ref, kind)).Set("kind", kind).Set("ied", ied.ID).
		Set("ref", ref).Set("address", address).Set("scale", scale).Set("description", name).
		Set("ied_name", ied.Name).Set("ied_role", ied.Role).Set("monitor_only", ied.Role != "simulated")
}

// nameOf is the name a new point gets: the typed one, else the SCL description, else the
// short reference ("XCBR1 Pos").
func nameOf(ied *IED, ref, typed string) string {
	if typed != "" {
		return head(typed, 120)
	}
	if description := ied.Description(ref); description != "" {
		return head(description, 120)
	}
	return ShortName(ref)
}

func head(text string, size int) string {
	runes := []rune(text)
	if len(runes) > size {
		return string(runes[:size])
	}
	return text
}

// checkWithEngine runs the saved-list rules (gateway.ValidatePoints) over the new points,
// so a planned point is never one the point list save would refuse.
func checkWithEngine(protocol string, results []Result, ieds []*pyjson.Object) {
	if !OK(results) {
		return
	}
	_, problems, err := gateway.ValidatePoints(protocol, Points(results), ieds)
	if err != nil {
		results[0].Problem = err.Error()
		return
	}
	for _, problem := range problems { // "Point <n> (...): reason", n counts the new points
		number := 0
		fmt.Sscanf(problem, "Point %d", &number)
		if number < 1 || number > len(results) {
			number = 1
		}
		if results[number-1].Problem == "" {
			results[number-1].Problem = problem
		}
	}
}
