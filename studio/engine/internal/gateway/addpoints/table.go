package addpoints

import (
	"encoding/csv"
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
)

// Row is one pasted or Excel row as text, by meaning (the columns were recognised).
type Row struct {
	Name, Type, IED, Point, Address, Scale string
}

// Columns are the meanings a column can have, in the template's order.
var Columns = []string{"name", "type", "ied", "point", "address", "scale"}

// JSON is the row as {name, type, ied, point, address, scale}.
func (row Row) JSON() *pyjson.Object {
	return pyjson.NewObject().Set("name", row.Name).Set("type", row.Type).Set("ied", row.IED).
		Set("point", row.Point).Set("address", row.Address).Set("scale", row.Scale)
}

func (row Row) empty() bool {
	return strings.TrimSpace(row.Name+row.Type+row.IED+row.Point+row.Address+row.Scale) == ""
}

func (row *Row) set(column, value string) {
	value = strings.TrimSpace(value)
	switch column {
	case "name":
		row.Name = value
	case "type":
		row.Type = value
	case "ied":
		row.IED = value
	case "point":
		row.Point = value
	case "address":
		row.Address = value
	case "scale":
		row.Scale = value
	}
}

// headerWords name each column as people title it in their SCADA lists.
var headerWords = map[string][]string{
	"address": {"address", "ioa", "index", "register", "coil", "addr", "information object address", "io address"},
	"type":    {"type", "kind", "type id", "typeid", "point type", "data type"},
	"ied":     {"ied", "device", "ied name", "bay", "source"},
	"point":   {"point", "ref", "reference", "path", "signal", "point ref", "object", "point reference", "iec 61850 reference"},
	"name":    {"name", "description", "desc", "label", "text", "point name", "signal name"},
	"scale":   {"scale", "factor", "multiplier"},
}

// headerColumn is the meaning of a header cell ("" when it is no known title).
func headerColumn(cell string) string {
	want := fold(cell)
	for column, words := range headerWords {
		for _, word := range words {
			if want == word {
				return column
			}
		}
	}
	return ""
}

const byteOrderMark = "\xef\xbb\xbf"

// SplitText turns pasted text into cells: Excel and LibreOffice copy rows as tab-separated
// lines; text without tabs is split on ";" or ",".
func SplitText(text string) [][]string {
	text = strings.TrimPrefix(strings.ReplaceAll(text, "\r\n", "\n"), byteOrderMark)
	delimiter := ','
	switch {
	case strings.Contains(text, "\t"):
		delimiter = '\t'
	case strings.Contains(text, ";"):
		delimiter = ';'
	}
	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma, reader.LazyQuotes, reader.FieldsPerRecord = delimiter, true, -1
	records, _ := reader.ReadAll()
	return records
}

// ReadTable recognises the columns of a table and reads its rows. With a header row its
// titles decide; without one each column is guessed from its first row (a number is the
// address, a known type is the type, a project IED name is the IED, a path with "/" is the
// point, other text is the name). numbers are the table's line numbers of the rows.
func ReadTable(cells [][]string, protocol string, index *Index) (rows []Row, numbers []int, columns []string) {
	if len(cells) == 0 {
		return nil, nil, nil
	}
	columns, header := make([]string, len(cells[0])), false
	for at, cell := range cells[0] {
		if columns[at] = headerColumn(cell); columns[at] != "" {
			header = true
		}
	}
	start := 0
	if header {
		start = 1
	} else {
		columns = guessColumns(cells[0], protocol, index)
	}
	for line := start; line < len(cells); line++ {
		row := Row{}
		for at, cell := range cells[line] {
			if at < len(columns) {
				row.set(columns[at], cell)
			}
		}
		if !row.empty() {
			rows, numbers = append(rows, row), append(numbers, line+1)
		}
	}
	return rows, numbers, columns
}

func guessColumns(first []string, protocol string, index *Index) []string {
	columns, used := make([]string, len(first)), map[string]bool{}
	claim := func(at int, column string) {
		if !used[column] {
			columns[at], used[column] = column, true
		}
	}
	for at, cell := range first {
		cell = strings.TrimSpace(cell)
		_, isType := ResolveType(protocol, cell)
		_, number := strconv.ParseInt(cell, 10, 64)
		switch {
		case cell == "":
		case number == nil:
			claim(at, "address")
		case isType:
			claim(at, "type")
		case index.Find(cell) != nil:
			claim(at, "ied")
		case strings.Contains(cell, "/"):
			claim(at, "point")
		default:
			claim(at, "name")
		}
	}
	return columns
}
