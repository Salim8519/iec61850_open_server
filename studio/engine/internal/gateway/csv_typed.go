package gateway

import "studio-engine/internal/pyjson"

// Set point formats in the point list CSV (ADR-0107, Go only). Like the own point columns,
// "format" and "word_order" are written only when the list has a set point, so every other
// list exports exactly as before, and a file without them imports exactly as before.
var typedColumns = []string{"format", "word_order"}

func hasFormats(points []any) bool {
	for _, value := range points {
		if point, ok := value.(*pyjson.Object); ok && get(point, "kind") == KindSetpoint {
			return true
		}
	}
	return false
}

// typedCells are the extra cells of one exported row ("" for a point without a format).
func typedCells(point *pyjson.Object) []any {
	cells := make([]any, len(typedColumns))
	for index, name := range typedColumns {
		text, _ := getOr(point, name, "").(string)
		cells[index] = text
	}
	return cells
}

// typedColumnsAt finds the extra columns in the header row (after the six standard ones).
func typedColumnsAt(header []string) map[string]int {
	found := map[string]int{}
	for index := len(csvColumns); index < len(header); index++ {
		for _, name := range typedColumns {
			if _, seen := found[name]; !seen && headerIs(header[index], name) {
				found[name] = index
			}
		}
	}
	return found
}

// applyTypedCells copies a set point's format cells; ValidatePoints checks them.
func applyTypedCells(point *pyjson.Object, row []string, at map[string]int) {
	if get(point, "kind") != KindSetpoint {
		return
	}
	for _, name := range typedColumns {
		if text := cellAt(row, at, name); text != "" {
			point.Set(name, text)
		}
	}
}
