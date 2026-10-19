// Package pycsv reads and writes CSV exactly as CPython 3.12's csv module does with the
// default "excel" dialect over io.StringIO text (lines split on "\n" only; a "\r" inside a
// line ends the record), including its error texts.
package pycsv

import (
	"strconv"
	"strings"
)

// FieldLimit is csv.field_size_limit() (131072 characters).
const FieldLimit = 131072

// Error is csv.Error; callers either turn it into a ValueError or let it surface as a
// server error, exactly where Python did.
type Error struct{ Text string }

func (e *Error) Error() string { return e.Text }

type state int

const (
	startRecord state = iota
	startField
	inField
	inQuotedField
	quoteInQuotedField
	eatCRNL
)

// Reader yields rows like csv.reader(io.StringIO(text), strict=Strict).
type Reader struct {
	Strict  bool
	text    string
	pos     int
	LineNum int
	state   state
	fields  []string
	field   []rune
}

// NewReader returns a reader over text.
func NewReader(text string, strict bool) *Reader { return &Reader{text: text, Strict: strict} }

// Next returns the next row; ok is false at the end of the input. A blank line is an empty row.
func (r *Reader) Next() (row []string, ok bool, err error) {
	r.fields, r.field, r.state = nil, nil, startRecord
	for {
		if r.pos >= len(r.text) {
			if len(r.field) != 0 || r.state == inQuotedField {
				if r.Strict {
					return nil, false, &Error{"unexpected end of data"}
				}
				r.save()
				return r.fields, true, nil
			}
			return nil, false, nil
		}
		line := r.nextLine()
		r.LineNum++
		for _, c := range line {
			if err := r.process(c, false); err != nil {
				return nil, false, err
			}
		}
		if err := r.process(0, true); err != nil {
			return nil, false, err
		}
		if r.state == startRecord {
			if r.fields == nil {
				r.fields = []string{}
			}
			return r.fields, true, nil
		}
	}
}

func (r *Reader) nextLine() string {
	end := strings.IndexByte(r.text[r.pos:], '\n')
	if end < 0 {
		line := r.text[r.pos:]
		r.pos = len(r.text)
		return line
	}
	line := r.text[r.pos : r.pos+end+1]
	r.pos += end + 1
	return line
}

func (r *Reader) save() {
	r.fields = append(r.fields, string(r.field))
	r.field = nil
}

func (r *Reader) add(c rune) error {
	if len(r.field) >= FieldLimit {
		return &Error{"field larger than field limit (" + strconv.Itoa(FieldLimit) + ")"}
	}
	r.field = append(r.field, c)
	return nil
}

func newline(c rune) bool { return c == '\n' || c == '\r' }

// process is _csv.c parse_process_char for the excel dialect; eol marks the end of a line.
func (r *Reader) process(c rune, eol bool) error {
	switch r.state {
	case startRecord:
		if eol {
			return nil
		}
		if newline(c) {
			r.state = eatCRNL
			return nil
		}
		r.state = startField
		return r.process(c, eol)
	case startField:
		switch {
		case eol || newline(c):
			r.save()
			r.state = endState(eol)
		case c == '"':
			r.state = inQuotedField
		case c == ',':
			r.save()
		default:
			r.state = inField
			return r.add(c)
		}
	case inField:
		switch {
		case eol || newline(c):
			r.save()
			r.state = endState(eol)
		case c == ',':
			r.save()
			r.state = startField
		default:
			return r.add(c)
		}
	case inQuotedField:
		switch {
		case eol:
		case c == '"':
			r.state = quoteInQuotedField
		default:
			return r.add(c)
		}
	case quoteInQuotedField:
		switch {
		case !eol && c == '"':
			r.state = inQuotedField
			return r.add(c)
		case !eol && c == ',':
			r.save()
			r.state = startField
		case eol || newline(c):
			r.save()
			r.state = endState(eol)
		case !r.Strict:
			r.state = inField
			return r.add(c)
		default:
			return &Error{"',' expected after '\"'"}
		}
	case eatCRNL:
		switch {
		case eol:
			r.state = startRecord
		case newline(c):
		default:
			return &Error{"new-line character seen in unquoted field - do you need to open the file with newline=''?"}
		}
	}
	return nil
}

func endState(eol bool) state {
	if eol {
		return startRecord
	}
	return eatCRNL
}

// ReadAll returns every row, stopping at the first error.
func ReadAll(text string, strict bool) ([][]string, error) {
	reader := NewReader(text, strict)
	var rows [][]string
	for {
		row, ok, err := reader.Next()
		if err != nil {
			return rows, err
		}
		if !ok {
			return rows, nil
		}
		rows = append(rows, row)
	}
}
