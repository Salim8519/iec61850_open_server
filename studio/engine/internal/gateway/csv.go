package gateway

import (
	"sort"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pycsv"
	"studio-engine/internal/tools/pytext"
)

// csvColumns are the point list CSV columns (gateway_csv.COLUMNS).
var csvColumns = []string{"type", "address", "ied", "reference", "scale", "description"}

const maxCSVBytes = 2 * 1024 * 1024

// ExportCSV is gateway_csv.export_csv(points, ieds): points sorted by type then address,
// IEDs written by name. Points are stored gateway points (str kind, int address); IED ids
// are str as the importer makes them.
func ExportCSV(points []any, ieds []*pyjson.Object) (string, error) {
	names := map[string]any{}
	for _, ied := range ieds {
		id, err := subscript(ied, "id")
		if err != nil {
			return "", err
		}
		name, err := subscript(ied, "name")
		if err != nil {
			return "", err
		}
		if err := hashable(id); err != nil {
			return "", err
		}
		if text, ok := id.(string); ok {
			names[text] = name
		}
	}
	type keyed struct {
		point   *pyjson.Object
		kind    string
		address any
	}
	rows := make([]keyed, len(points))
	for index, value := range points {
		kind, err := subscript(value, "kind")
		if err != nil {
			return "", err
		}
		address, err := subscript(value, "address")
		if err != nil {
			return "", err
		}
		text, isText := kind.(string)
		if !isText || !intLike(address) {
			return "", crash("TypeError", "'<' not supported between instances of '"+typeName(kind)+
				"' and '"+typeName(address)+"'") // unreachable for stored (normalized) points
		}
		rows[index] = keyed{value.(*pyjson.Object), text, address}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].kind != rows[j].kind {
			return rows[i].kind < rows[j].kind
		}
		return cmpInt(rows[i].address, rows[j].address) < 0
	})
	writer := pycsv.NewWriter()
	own := hasOwn(points)
	if own {
		writer.Row(append(anyStrings(csvColumns), anyStrings(ownColumns)...)...)
	} else {
		writer.Row(anyStrings(csvColumns)...)
	}
	for _, row := range rows {
		ied, err := subscript(row.point, "ied")
		if err != nil {
			return "", err
		}
		if err := hashable(ied); err != nil {
			return "", err
		}
		name := ied
		if text, ok := ied.(string); ok {
			if found, known := names[text]; known {
				name = found
			}
		}
		ref, err := subscript(row.point, "ref")
		if err != nil {
			return "", err
		}
		scale, err := formatG(getOr(row.point, "scale", 1.0))
		if err != nil {
			return "", err
		}
		cells := []any{row.kind, row.address, name, ref, scale, getOr(row.point, "description", "")}
		if own {
			cells = append(cells, ownCells(row.point)...)
		}
		writer.Row(cells...)
	}
	return writer.String(), nil
}

// headerIs is cell.strip().lower() == word for an ASCII word: only ASCII letters and the
// Kelvin sign (lowered to "k") lower to ASCII in Python.
func headerIs(cell, word string) bool {
	var lowered strings.Builder
	for _, r := range pytext.Strip(cell) {
		switch {
		case r < 0x80:
			if r >= 'A' && r <= 'Z' {
				r += 'a' - 'A'
			}
			lowered.WriteRune(r)
		case r == 0x212a:
			lowered.WriteByte('k')
		default:
			return false
		}
	}
	return lowered.String() == word
}

// ImportCSV is gateway_csv.import_csv(text, ieds): rows matched to IEDs by name; nothing is
// invented. A csv.Error is returned as a *pycsv.Error (Python answered 500).
func ImportCSV(text string, ieds []*pyjson.Object) ([]any, []string, error) {
	if len(text) > maxCSVBytes {
		return nil, nil, valueError("The CSV file is larger than 2 MB")
	}
	rows, err := pycsv.ReadAll(strings.TrimLeft(text, "\xef\xbb\xbf"), false)
	if err != nil {
		return nil, nil, err
	}
	if len(rows) == 0 || !headerMatches(rows[0]) {
		return nil, nil, valueError("The first row must be: " + strings.Join(csvColumns, ","))
	}
	byName := map[string]any{}
	for _, ied := range ieds {
		name, err := subscript(ied, "name")
		if err != nil {
			return nil, nil, err
		}
		id, err := subscript(ied, "id")
		if err != nil {
			return nil, nil, err
		}
		if err := hashable(name); err != nil {
			return nil, nil, err
		}
		if text, ok := name.(string); ok {
			byName[text] = id
		}
	}
	points, problems, extra := []any{}, []string{}, ownColumnsAt(rows[0])
	for index, row := range rows[1:] {
		line := "Line " + itoa(index+2)
		cells := make([]string, len(csvColumns))
		blank := true
		for column, cell := range row {
			cell = pytext.Strip(cell)
			blank = blank && cell == ""
			if column < len(cells) {
				cells[column] = cell
			}
		}
		if blank {
			continue
		}
		kind, address, iedName, reference, scale, description := cells[0], cells[1], cells[2], cells[3], cells[4], cells[5]
		if !contains(Kinds, kind) {
			problems = append(problems, line+": unknown type '"+kind+"'. Use one of: "+strings.Join(Kinds, ", "))
			continue
		}
		if isOwnRow(row, extra) { // Go-only: an own point names no IED
			point, ok := ownRow(kind, address, scale, description, row, extra)
			if !ok {
				problems = append(problems, line+": address must be a whole number, scale and value numbers")
				continue
			}
			points = append(points, point)
			continue
		}
		id, known := byName[iedName]
		if !known {
			problems = append(problems, line+": no IED named '"+iedName+"' in this project")
			continue
		}
		number, err := parseInt(address)
		factor := 1.0
		if err == nil && scale != "" {
			factor, err = pyFloat(scale)
		}
		if err != nil {
			problems = append(problems, line+": address must be a whole number and scale a number")
			continue
		}
		points = append(points, pyjson.NewObject().Set("kind", kind).Set("address", number).Set("ied", id).
			Set("ref", reference).Set("scale", factor).Set("description", description))
	}
	return points, problems, nil
}

func headerMatches(row []string) bool {
	if len(row) < 4 {
		return false
	}
	for index, word := range csvColumns[:4] {
		if !headerIs(row[index], word) {
			return false
		}
	}
	return true
}
