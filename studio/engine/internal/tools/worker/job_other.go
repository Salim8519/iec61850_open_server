//go:build !windows

package worker

import "errors"

// adopt has no Job Object outside Windows; the product ships for Windows only.
func adopt(pid int) error { return errors.New("job objects need Windows") }
