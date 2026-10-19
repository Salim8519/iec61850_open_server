//go:build !windows

package netcfg

import "studio-engine/internal/apperr"

// UACElevator exists only on Windows.
type UACElevator struct{}

// Elevate refuses off Windows, as elevation.run_elevated does.
func (UACElevator) Elevate(executable string, arguments []string) (uint32, error) {
	return 0, apperr.RuntimeError("Lab network setup is only available on Windows")
}
