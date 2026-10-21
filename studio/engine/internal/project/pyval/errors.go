package pyval

// Error is a Python exception other than StudioError, ValueError, KeyError and
// RuntimeError (those use apperr): TypeError, AttributeError or OverflowError raised by
// CPython itself on malformed input. Flask answers them as an internal error (500).
type Error struct {
	Class   string
	Message string
}

func (e *Error) Error() string { return e.Message }

// TypeError returns a TypeError with message.
func TypeError(message string) *Error { return &Error{Class: "TypeError", Message: message} }

// AttributeError returns an AttributeError with message.
func AttributeError(message string) *Error {
	return &Error{Class: "AttributeError", Message: message}
}

// OverflowError returns an OverflowError with message.
func OverflowError(message string) *Error {
	return &Error{Class: "OverflowError", Message: message}
}

// Unhashable is the TypeError of hashing a list or dict (set or dict membership).
func Unhashable(v any) *Error { return TypeError("unhashable type: '" + TypeName(v) + "'") }

// NotMapping is the TypeError of `{**v}` when v is not a dict.
func NotMapping(v any) *Error { return TypeError("'" + TypeName(v) + "' object is not a mapping") }

// NotIterable is the TypeError of iterating a number, bool or None.
func NotIterable(v any) *Error { return TypeError("'" + TypeName(v) + "' object is not iterable") }

// NoAttribute is the AttributeError of calling a dict method on another type.
func NoAttribute(v any, name string) *Error {
	return AttributeError("'" + TypeName(v) + "' object has no attribute '" + name + "'")
}

// CheckHashable returns the TypeError Python raises when v is hashed.
func CheckHashable(v any) error {
	if Hashable(v) {
		return nil
	}
	return Unhashable(v)
}
