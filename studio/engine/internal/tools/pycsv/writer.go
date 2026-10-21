package pycsv

import (
	"strings"

	"studio-engine/internal/tools/pytext"
)

// Writer is csv.writer(stream, lineterminator=Terminator) with QUOTE_MINIMAL.
type Writer struct {
	out        strings.Builder
	Terminator string
}

// NewWriter returns a writer; Python's default line terminator is "\r\n".
func NewWriter() *Writer { return &Writer{Terminator: "\r\n"} }

// Row writes one record. Values follow csv.writer: None is "", other values str().
func (w *Writer) Row(values ...any) {
	for index, value := range values {
		if index > 0 {
			w.out.WriteByte(',')
		}
		text := ""
		if value != nil {
			text = pytext.Str(value)
		}
		quote := strings.ContainsAny(text, ",\"\r\n") || strings.ContainsAny(text, w.Terminator)
		if len(values) == 1 && text == "" {
			quote = true // a record of one empty field is written as "" so it reads back
		}
		if quote {
			w.out.WriteByte('"')
			w.out.WriteString(strings.ReplaceAll(text, `"`, `""`))
			w.out.WriteByte('"')
		} else {
			w.out.WriteString(text)
		}
	}
	w.out.WriteString(w.Terminator)
}

// String returns everything written so far (io.StringIO.getvalue()).
func (w *Writer) String() string { return w.out.String() }
