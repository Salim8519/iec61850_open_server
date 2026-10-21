//go:build !windows

package licensing

import "errors"

// readMachineParts fails outside Windows, as hwid.machine_id does.
func readMachineParts() (MachineParts, error) {
	return MachineParts{}, errors.New("Machine licensing is available on Windows only")
}

// readRawTrialParts finds no part outside Windows, as trial_parts.machine_parts does.
func readRawTrialParts() RawTrialParts { return RawTrialParts{} }
