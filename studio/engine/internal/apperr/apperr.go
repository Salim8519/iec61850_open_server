// Package apperr carries the errors Python raised, so httpapi can answer with the same
// status and JSON body Flask did (parity inventory: http-conventions.md).
package apperr

// Kind names the Python exception class an error stands for.
type Kind int

const (
	// Studio is StudioError: 400, or 409 when Code is "conflict"; other codes as Python maps them.
	Studio Kind = iota
	// Value is ValueError: 400 with Python's message text.
	Value
	// Key is KeyError: 400 with Python's repr of the key, e.g. 'ied_id' with quotes.
	Key
	// Runtime is RuntimeError, TimeoutError or native failures: 503.
	Runtime
)

// Error is an error with the Python class, message and optional code and details.
type Error struct {
	Kind    Kind
	Message string
	Code    string
	Details any
}

func (e *Error) Error() string { return e.Message }

// New returns a StudioError with message and Python's default code "validation"; Flask
// answers {"code", "details", "error"} with 400 (409 for code "conflict").
func New(message string) *Error { return &Error{Kind: Studio, Message: message, Code: "validation"} }

// Coded returns a StudioError with a code such as "conflict" or "not_found".
func Coded(message, code string) *Error { return &Error{Kind: Studio, Message: message, Code: code} }

// ValueError returns a ValueError with message.
func ValueError(message string) *Error { return &Error{Kind: Value, Message: message} }

// KeyError returns a KeyError for a plain ASCII key such as 'ied_id'. For any key that may
// hold quotes or non-ASCII text use pycompat.KeyError, which applies Python's full repr.
func KeyError(key string) *Error { return &Error{Kind: Key, Message: "'" + key + "'"} }

// RuntimeError returns a RuntimeError with message.
func RuntimeError(message string) *Error { return &Error{Kind: Runtime, Message: message} }
