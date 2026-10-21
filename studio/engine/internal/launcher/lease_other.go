//go:build !windows

package launcher

import "errors"

// Lease is unsupported outside Windows (ScadaBench is a Windows application).
type Lease struct{}

// AcquireLease always fails outside Windows.
func AcquireLease(directory string) (*Lease, error) {
	return nil, errors.New("the studio.lock lease needs Windows")
}

// Release does nothing.
func (l *Lease) Release() {}
