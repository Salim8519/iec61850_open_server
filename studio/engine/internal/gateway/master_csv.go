package gateway

import (
	"unicode"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pycsv"
	"studio-engine/internal/tools/pytext"
)

// ExportMasterCSV is master_points.export_csv(display_rows) for Rows output.
func ExportMasterCSV(rows []any) string {
	writer := pycsv.NewWriter()
	writer.Row("group", "address", "label", "value", "quality", "watched")
	for _, value := range rows {
		row := value.(*pyjson.Object)
		watched := "no"
		if pytext.Truthy(get(row, "ticked")) {
			watched = "yes"
		}
		writer.Row(get(row, "group"), get(row, "address"), get(row, "label"), get(row, "value_text"),
			get(row, "quality_text"), watched)
	}
	return writer.String()
}

// digitOnly are the characters str.isdigit accepts beyond Unicode decimals (Nd), from
// CPython 3.12's Unicode 15.0 database (superscripts, circled and other digit forms).
var digitOnly = [][2]rune{{0xB2, 0xB3}, {0xB9, 0xB9}, {0x1369, 0x1371}, {0x19DA, 0x19DA}, {0x2070, 0x2070},
	{0x2074, 0x2079}, {0x2080, 0x2089}, {0x2460, 0x2468}, {0x2474, 0x247C}, {0x2488, 0x2490},
	{0x24EA, 0x24EA}, {0x24F5, 0x24FD}, {0x24FF, 0x24FF}, {0x2776, 0x277E}, {0x2780, 0x2788},
	{0x278A, 0x2792}, {0x10A40, 0x10A43}, {0x10E60, 0x10E68}, {0x11052, 0x1105A}, {0x1F100, 0x1F10A}}

// isDigits is str.isdigit(): at least one character, every one a digit.
func isDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if unicode.Is(unicode.Nd, r) {
			continue
		}
		found := false
		for _, span := range digitOnly {
			found = found || (r >= span[0] && r <= span[1])
		}
		if !found {
			return false
		}
	}
	return true
}

// labelColumns are the columns import_labels needs.
var labelColumns = []string{"group", "address", "label"}

// columnOf is the label column a header name stands for after strip().lower(), or "".
func columnOf(name string) string {
	for _, column := range labelColumns {
		if headerIs(name, column) {
			return column
		}
	}
	return ""
}

// ImportLabels is import_labels(text): a CSV with group, address and label columns read by
// csv.DictReader -> ({key: label}, problems). A csv.Error is returned as *pycsv.Error and
// int() of a digit that is no decimal (such as "²") gives Python's ValueError.
func ImportLabels(text string) (*pyjson.Object, []string, error) {
	labels, problems := pyjson.NewObject(), []string{}
	reader := pycsv.NewReader(text, false)
	fieldnames, ok, err := reader.Next()
	if err != nil {
		return nil, nil, err
	}
	present := map[string]bool{}
	for _, name := range fieldnames {
		present[columnOf(name)] = true
	}
	if !ok || len(fieldnames) == 0 || !present["group"] || !present["address"] || !present["label"] {
		return labels, []string{"The CSV needs the columns group, address and label (export one to see the layout)"}, nil
	}
	for number := 2; ; number++ {
		row, err := nextRecord(reader)
		if err != nil {
			return nil, nil, err
		}
		if row == nil {
			return labels, problems, nil
		}
		cells := dictRow(fieldnames, row)
		group, address, label := cells["group"], cells["address"], cells["label"]
		if !contains(groups, group) || !isDigits(address) {
			problems = append(problems, "Line "+itoa(number)+": unknown group '"+group+"' or address '"+address+"'")
			continue
		}
		if label == "" {
			continue
		}
		value, err := parseInt(address)
		if err != nil {
			return nil, nil, err
		}
		labels.Set(group+":"+pytext.Str(value), pytext.Head(label, 80))
	}
}

// nextRecord is DictReader's next row: blank records are skipped; nil at the end.
func nextRecord(reader *pycsv.Reader) ([]string, error) {
	for {
		row, ok, err := reader.Next()
		if err != nil || !ok {
			return nil, err
		}
		if len(row) > 0 {
			return row, nil
		}
	}
}

// dictRow is {name.strip().lower(): (value or "").strip() for name, value in dict(zip(...))
// if name} for the three label columns: missing cells are None (""), and a name repeated
// after the row's end resets that column to None as DictReader's restval does.
func dictRow(fieldnames, row []string) map[string]string {
	values := map[string]*string{}
	order := []string{}
	for index, name := range fieldnames {
		if _, seen := values[name]; !seen {
			order = append(order, name)
		}
		if index < len(row) {
			cell := row[index]
			values[name] = &cell
		} else {
			values[name] = nil
		}
	}
	cells := map[string]string{}
	for _, name := range order {
		if name == "" {
			continue
		}
		if column := columnOf(name); column != "" {
			cells[column] = ""
			if values[name] != nil {
				cells[column] = pytext.Strip(*values[name])
			}
		}
	}
	return cells
}
