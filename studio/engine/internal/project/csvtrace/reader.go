// Package csvtrace parses measurement traces exactly as domain/csv_trace.py does with
// CPython 3.12's csv module (excel dialect, strict=True) over io.StringIO lines.
package csvtrace

import (
	"errors"
	"strings"
)

// errMalformed stands for csv.Error; read_rows reports every csv.Error the same way.
var errMalformed = errors.New("csv error")

const fieldLimit = 131072 // csv.field_size_limit() default

type parserState int

const (
	startRecord parserState = iota
	startField
	inField
	inQuotedField
	quoteInQuotedField
	eatCRNL
)

// eol marks the end of one physical line, as _csv.c does after each line.
const eol = rune(-2)

// reader is a port of _csv.c's Reader for the excel dialect with strict=True.
type reader struct {
	lines  []string
	state  parserState
	fields []string
	field  []rune
}

// newReader splits text like iterating io.StringIO(text): lines end after each "\n".
func newReader(text string) *reader {
	r := &reader{}
	for text != "" {
		index := strings.IndexByte(text, '\n')
		if index < 0 {
			r.lines = append(r.lines, text)
			break
		}
		r.lines = append(r.lines, text[:index+1])
		text = text[index+1:]
	}
	return r
}

// next returns the next record; done is true at the end of input.
func (r *reader) next() (record []string, done bool, err error) {
	r.fields, r.field, r.state = []string{}, r.field[:0], startRecord
	for {
		if len(r.lines) == 0 {
			if len(r.field) != 0 || r.state == inQuotedField {
				return nil, false, errMalformed // "unexpected end of data"
			}
			return nil, true, nil
		}
		line := r.lines[0]
		r.lines = r.lines[1:]
		for _, c := range line {
			if err := r.process(c); err != nil {
				return nil, false, err
			}
		}
		if err := r.process(eol); err != nil {
			return nil, false, err
		}
		if r.state == startRecord {
			return r.fields, false, nil
		}
	}
}

func (r *reader) save() {
	r.fields = append(r.fields, string(r.field))
	r.field = r.field[:0]
}

func (r *reader) add(c rune) error {
	if len(r.field) >= fieldLimit {
		return errMalformed // "field larger than field limit (131072)"
	}
	r.field = append(r.field, c)
	return nil
}

func lineEnd(c rune) bool { return c == '\n' || c == '\r' || c == eol }

func (r *reader) endRecord(c rune) {
	r.save()
	if c == eol {
		r.state = startRecord
	} else {
		r.state = eatCRNL
	}
}

func (r *reader) process(c rune) error {
	switch r.state {
	case startRecord:
		if c == eol {
			return nil
		}
		if c == '\n' || c == '\r' {
			r.state = eatCRNL
			return nil
		}
		r.state = startField
		return r.process(c)
	case startField:
		switch {
		case lineEnd(c):
			r.endRecord(c)
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
		case lineEnd(c):
			r.endRecord(c)
		case c == ',':
			r.save()
			r.state = startField
		default:
			return r.add(c)
		}
	case inQuotedField:
		switch {
		case c == eol:
		case c == '"':
			r.state = quoteInQuotedField
		default:
			return r.add(c)
		}
	case quoteInQuotedField:
		switch {
		case c == '"':
			r.state = inQuotedField
			return r.add(c)
		case c == ',':
			r.save()
			r.state = startField
		case lineEnd(c):
			r.endRecord(c)
		default:
			return errMalformed // "',' expected after '\"'"
		}
	case eatCRNL:
		switch {
		case c == '\n' || c == '\r':
		case c == eol:
			r.state = startRecord
		default:
			return errMalformed // "new-line character seen in unquoted field"
		}
	}
	return nil
}
