//go:build !windows

package licensing

import "errors"

// readMachineParts fails outside Windows, as hwid.machine_id does.
func readMachineParts(bool) (MachineParts, error) {
	return MachineParts{}, errors.New("Machine licensing is available on Windows only")
}
