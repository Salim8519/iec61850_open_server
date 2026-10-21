//go:build !windows

package hostkey

import "errors"

type unavailable struct{}

// System has no DPAPI off Windows: the host key cannot be kept, so no key is ever offered.
func System() Protector { return unavailable{} }

func (unavailable) Protect([]byte) ([]byte, error)   { return nil, errors.New("DPAPI needs Windows") }
func (unavailable) Unprotect([]byte) ([]byte, error) { return nil, ErrUnprotect }
