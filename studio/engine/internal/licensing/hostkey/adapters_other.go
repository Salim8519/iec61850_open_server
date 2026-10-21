//go:build !windows

package hostkey

import "errors"

// SystemAdapters needs the Windows IP Helper API.
func SystemAdapters() ([]Adapter, error) { return nil, errors.New("listing adapters needs Windows") }
