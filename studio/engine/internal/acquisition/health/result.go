// Package health builds the station health view: one verdict per IED and for the station,
// from SCD data and what the live sessions and the GOOSE listener already observed, plus
// its CSV and HTML exports (ports of domain/goose_consistency.py, health_model_checks.py,
// station_health.py and health_export.py).
//
// Inputs and outputs are pyjson trees with Python dict/list semantics (see package pyv).
package health

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// Result is goose_consistency.result: one check row {area, item, result, message,
// observed_at, source, detail}. Python's defaults are observedAt nil, source "scd" and
// detail nil; a nil or empty detail becomes a new empty object.
func Result(area string, item any, verdict, message string, observedAt any, source string,
	detail *pyjson.Object) *pyjson.Object {
	return row(area, item, verdict, message, observedAt, source, detail)
}

// row is Result with a message of any type: Python passes some values through unchanged
// (`listener.get("message") or ...`).
func row(area string, item any, verdict string, message, observedAt any, source string,
	detail *pyjson.Object) *pyjson.Object {
	if detail == nil || detail.Len() == 0 {
		detail = pyjson.NewObject()
	}
	return pyv.O("area", area, "item", item, "result", verdict, "message", message,
		"observed_at", observedAt, "source", source, "detail", detail)
}

// notChecked is result(area, item, "not_checked", message) with Python's defaults.
func notChecked(area string, item any, message any) *pyjson.Object {
	return row(area, item, "not_checked", message, nil, "scd", nil)
}

// Sentence is sentence: findings joined with "; ", first character upper-cased, and a
// final full stop, without changing the case of references.
func Sentence(parts []string) string {
	return upperFirst(strings.Join(parts, "; ")) + "."
}

// upperFirst is text[:1].upper() + text[1:]. Python's upper() uses full case mappings;
// Go maps one rune to one rune, so the ~100 characters whose upper case has several
// characters (ß -> SS) differ. Callers only pass ASCII first letters.
func upperFirst(text string) string {
	first, size := utf8.DecodeRuneInString(text)
	if size == 0 {
		return text
	}
	return string(unicode.ToUpper(first)) + text[size:]
}

// filled is Python truthiness of an object that may be missing (nil): non-empty.
func filled(object *pyjson.Object) bool {
	return object != nil && object.Len() > 0
}

// pyLen is len(value) for a list, string (code points) or object; 0 for anything else.
func pyLen(value any) int {
	switch v := value.(type) {
	case []any:
		return len(v)
	case string:
		return utf8.RuneCountInString(v)
	case *pyjson.Object:
		if v != nil {
			return v.Len()
		}
	}
	return 0
}
