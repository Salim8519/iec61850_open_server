package pycompat

import "studio-engine/internal/apperr"

// KeyError returns Python's KeyError for key: its text is repr(key), so a key holding a
// single quote is written in double quotes ("it's"), exactly as Flask answered.
func KeyError(key string) *apperr.Error {
	return &apperr.Error{Kind: apperr.Key, Message: Repr(key)}
}
