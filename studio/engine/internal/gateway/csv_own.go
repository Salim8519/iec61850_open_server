package gateway

import (
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Own points in the point list CSV (lane G, Go-only). The extra columns are written only
// when the list has own points, so a list of IED points exports exactly as before, and a
// file without them imports exactly as before.
var ownColumns = []string{"source", "value", "id", "target"}

func hasOwn(points []any) bool {
	for _, value := range points {
		if point, ok := value.(*pyjson.Object); ok && IsOwn(point) {
			return true
		}
	}
	return false
}

// ownCells are the extra cells of one exported row ("" for an IED point).
func ownCells(point *pyjson.Object) []any {
	if !IsOwn(point) {
		return []any{"", "", "", ""}
	}
	value := ""
	if stored := getOr(point, "value", nil); stored != nil {
		value = pytext.Str(stored)
	}
	return []any{OwnSource, value, pytext.Str(getOr(point, "id", "")), pytext.Str(getOr(point, "target", ""))}
}

// ownColumnsAt finds the extra columns in the header row (after the six standard ones).
func ownColumnsAt(header []string) map[string]int {
	found := map[string]int{}
	for index := len(csvColumns); index < len(header); index++ {
		for _, name := range ownColumns {
			if _, seen := found[name]; !seen && headerIs(header[index], name) {
				found[name] = index
			}
		}
	}
	return found
}

func cellAt(row []string, at map[string]int, name string) string {
	index, found := at[name]
	if !found || index >= len(row) {
		return ""
	}
	return pytext.Strip(row[index])
}

// isOwnRow: the row's source column says "manual" (or "own").
func isOwnRow(row []string, at map[string]int) bool {
	source := strings.ToLower(cellAt(row, at, "source"))
	return source == OwnSource || source == "own"
}

// ownRow reads an own point row; ValidatePoints checks the value against the type.
func ownRow(kind, address, scale, description string, row []string, at map[string]int) (*pyjson.Object, bool) {
	number, err := parseInt(address)
	factor := 1.0
	if err == nil && scale != "" {
		factor, err = pyFloat(scale)
	}
	if err != nil {
		return nil, false
	}
	point := pyjson.NewObject().Set("kind", kind).Set("source", OwnSource).Set("address", number).
		Set("ied", "").Set("ref", "").Set("scale", factor).Set("description", description)
	switch text := strings.ToLower(cellAt(row, at, "value")); text {
	case "":
	case "on", "true", "closed", "close":
		point.Set("value", true)
	case "off", "false", "open":
		point.Set("value", false)
	default:
		if whole, err := parseInt(text); err == nil {
			point.Set("value", whole)
		} else if real, err := pyFloat(text); err == nil {
			point.Set("value", real)
		} else {
			return nil, false
		}
	}
	for _, name := range []string{"id", "target"} {
		if text := cellAt(row, at, name); text != "" {
			point.Set(name, text)
		}
	}
	return point, true
}
