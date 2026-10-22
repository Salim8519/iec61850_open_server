package masterpoints

import (
	"strconv"
	"strings"

	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/pyjson"
)

// Columns are the meanings a column of a master sheet can have, in the export's order.
var Columns = []string{"name", "address", "type", "scale", "offset", "unit", "select", "value", "word_order"}

// headerWords name each column as people title it in their SCADA point lists.
var headerWords = map[string][]string{
	"name":       {"name", "description", "desc", "label", "text", "point name", "signal name", "signal"},
	"address":    {"address", "ioa", "index", "register", "coil", "addr", "information object address", "io address"},
	"type":       {"type", "kind", "type id", "typeid", "point type", "asdu type", "object"},
	"scale":      {"scale", "factor", "multiplier", "gain"},
	"offset":     {"offset"},
	"unit":       {"unit", "units", "eng unit", "engineering unit"},
	"select":     {"sbo", "select", "select before operate"},
	"value":      {"value", "command value", "default value", "step"},
	"word_order": {"word order", "word_order", "byte order", "swap"},
}

func headerColumn(cell string) string {
	want := strings.ToLower(strings.Join(strings.Fields(cell), " "))
	for column, words := range headerWords {
		for _, word := range words {
			if want == word {
				return column
			}
		}
	}
	return ""
}

// ReadTable recognises the columns of pasted or Excel cells and reads the rows as {column:
// text}. With a title row its words decide; without one a whole number is the address, a known
// type is the type and other text is the name. numbers are the rows' lines in the source.
func ReadTable(cells [][]string, protocol string) (rows []map[string]string, numbers []int, columns []string) {
	if len(cells) == 0 {
		return nil, nil, nil
	}
	columns, header := make([]string, len(cells[0])), false
	for at, cell := range cells[0] {
		if columns[at] = headerColumn(cell); columns[at] != "" {
			header = true
		}
	}
	start := 1
	if !header {
		start, columns = 0, guessColumns(cells[0], protocol)
	}
	for line := start; line < len(cells); line++ {
		row, filled := map[string]string{}, false
		for at, cell := range cells[line] {
			if at < len(columns) && columns[at] != "" {
				row[columns[at]] = strings.TrimSpace(cell)
				filled = filled || row[columns[at]] != ""
			}
		}
		if filled {
			rows, numbers = append(rows, row), append(numbers, line+1)
		}
	}
	return rows, numbers, columns
}

func guessColumns(first []string, protocol string) []string {
	columns, used := make([]string, len(first)), map[string]bool{}
	for at, cell := range first {
		cell = strings.TrimSpace(cell)
		_, isType := addpoints.ResolveMasterType(protocol, cell)
		_, notNumber := strconv.ParseInt(strings.ReplaceAll(cell, " ", ""), 10, 64)
		column := "name"
		switch {
		case cell == "":
			continue
		case notNumber == nil:
			column = "address"
		case isType:
			column = "type"
		}
		if !used[column] {
			columns[at], used[column] = column, true
		}
	}
	return columns
}

// CheckRows turns read rows into new points and checks every one before anything is added: a
// type of this protocol, an address that is a free whole number, numbers where numbers belong.
func CheckRows(protocol string, rows []map[string]string, numbers []int, current []any) []Result {
	plan, results := newPlanner(protocol, current), make([]Result, 0, len(rows))
	for at, row := range rows {
		raw, problem := rowPoint(protocol, row)
		var point *pyjson.Object
		if problem == "" {
			point, problem = plan.add(raw)
		}
		if point == nil {
			point = raw
		}
		results = append(results, Result{Number: numbers[at], Point: point, Problem: rowProblem(numbers[at], problem),
			Cells: cellsJSON(row)})
	}
	return results
}

// rowPoint reads the texts of one row into a raw point for CheckPoint.
func rowPoint(protocol string, row map[string]string) (*pyjson.Object, string) {
	raw := pyjson.NewObject().Set("name", row["name"])
	if row["type"] == "" {
		return raw, "choose a type"
	}
	entry, known := addpoints.ResolveMasterType(protocol, row["type"])
	if !known {
		return raw, "\"" + row["type"] + "\" is not " + article(protocol) + " " + ProtocolName(protocol) + " type; use one of: " +
			typeNames(protocol)
	}
	raw.Set("type", entry.Key)
	address, err := strconv.ParseInt(strings.ReplaceAll(row["address"], " ", ""), 10, 64)
	if err != nil {
		return raw, "write the " + AddressWord[protocol] + " the device uses (a whole number)"
	}
	raw.Set("address", address)
	for _, key := range []string{"scale", "offset", "value"} {
		if text := row[key]; text != "" {
			value, err := strconv.ParseFloat(strings.ReplaceAll(text, ",", "."), 64)
			if err != nil {
				return raw, "the " + key + " must be a number"
			}
			raw.Set(key, value)
		}
	}
	raw.Set("unit", row["unit"])
	// The gateway CSV writes low_first for the same order (0178: one meaning, both spellings read).
	if order := strings.ToLower(row["word_order"]); strings.HasPrefix(order, "little") || order == "low word first" || order == "low_first" {
		raw.Set("word_order", "little")
	}
	switch strings.ToLower(row["select"]) {
	case "no", "false", "0", "direct", "off":
		raw.Set("select", false)
	case "yes", "true", "1", "sbo", "on":
		raw.Set("select", true)
	}
	return raw, ""
}

func typeNames(protocol string) string {
	names := []string{}
	for _, entry := range addpoints.MasterTypes(protocol) {
		if entry.MostUsed {
			names = append(names, entry.Name)
		}
	}
	return strings.Join(names, "; ") + " (or see the full type list)" // the names hold commas
}

// article is "an IEC 104" but "a DNP3" and "a Modbus".
func article(protocol string) string {
	if protocol == "iec104" {
		return "an"
	}
	return "a"
}

func cellsJSON(row map[string]string) *pyjson.Object {
	cells := pyjson.NewObject()
	for _, column := range Columns {
		cells.Set(column, row[column])
	}
	return cells
}

// PlanText reads pasted text (tabs from Excel, or ";"/"," separated, a CSV file too).
func PlanText(protocol, text string, current []any) ([]Result, []string) {
	rows, numbers, columns := ReadTable(addpoints.SplitText(text), protocol)
	return CheckRows(protocol, rows, numbers, current), columns
}
