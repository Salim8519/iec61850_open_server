package signal

import "errors"

// errCSV stands for csv.Error: validate_trace turns any of them into "CSV is empty or malformed".
var errCSV = errors.New("csv error")

// pyCSVRows parses text like Python's csv.reader(io.StringIO(text), strict=True) with the
// excel dialect (delimiter ',', quotechar '"', doublequote, no escape, no skipinitialspace).
// It is a port of the state machine in CPython's Modules/_csv.c.
func pyCSVRows(text string) ([][]string, error) {
	const (
		startRecord = iota
		startField
		inField
		inQuotedField
		quoteInQuotedField
		eatCRNL
	)
	var rows [][]string
	runes := []rune(text)
	position := 0
	for position < len(runes) {
		var fields []string
		var field []rune
		state := startRecord
		saveField := func() { fields = append(fields, string(field)); field = field[:0] }
		lineDone := false
		for !lineDone {
			if position >= len(runes) {
				// End of input inside a line: StringIO yields the last line without a newline.
				if state == inQuotedField {
					return nil, errCSV // "unexpected end of data" in strict mode
				}
				if state != startRecord && state != eatCRNL {
					saveField()
				}
				lineDone = true
				break
			}
			c := runes[position]
			// StringIO splits lines at \n only; a \r inside a line ends the record in _csv.
			position++
			switch state {
			case startRecord:
				if c == '\n' || c == '\r' {
					state = eatCRNL
					if c == '\n' {
						lineDone = true
					}
					continue
				}
				state = startField
				fallthrough
			case startField:
				switch c {
				case '\n', '\r':
					saveField()
					state = eatCRNL
					if c == '\n' {
						lineDone = true
					}
				case '"':
					state = inQuotedField
				case ',':
					saveField()
				default:
					field = append(field, c)
					state = inField
				}
			case inField:
				switch c {
				case '\n', '\r':
					saveField()
					state = eatCRNL
					if c == '\n' {
						lineDone = true
					}
				case ',':
					saveField()
					state = startField
				default:
					field = append(field, c)
				}
			case inQuotedField:
				if c == '"' {
					state = quoteInQuotedField
				} else {
					field = append(field, c)
				}
			case quoteInQuotedField:
				switch c {
				case '"':
					field = append(field, c)
					state = inQuotedField
				case ',':
					saveField()
					state = startField
				case '\n', '\r':
					saveField()
					state = eatCRNL
					if c == '\n' {
						lineDone = true
					}
				default:
					return nil, errCSV // "',' expected after '\"'"
				}
			case eatCRNL:
				if c == '\n' {
					lineDone = true
				} else if c != '\r' {
					return nil, errCSV // "new-line character seen in unquoted field"
				}
			}
		}
		if fields == nil {
			fields = []string{}
		}
		rows = append(rows, fields)
	}
	return rows, nil
}
