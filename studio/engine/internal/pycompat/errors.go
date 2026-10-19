package pycompat

// Error is a Python exception that is not a ValueError, such as OverflowError, OSError or
// TypeError. Flask had no handler for these, so the Python engine answered 500; httpapi
// does the same for every error it does not map.
type Error struct {
	Class   string
	Message string
}

func (e *Error) Error() string { return e.Message }

func overflow(message string) error { return &Error{Class: "OverflowError", Message: message} }
