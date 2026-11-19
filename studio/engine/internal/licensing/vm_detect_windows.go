//go:build windows

package licensing

import "sync"

var (
	virtualMachineOnce sync.Once
	virtualMachine     bool
)

// IsVirtualMachine reports whether this computer's firmware is that of a virtual machine
// (SMBIOSIsVirtualMachine on the raw SMBIOS table, read once: firmware does not change while running).
func IsVirtualMachine() bool {
	virtualMachineOnce.Do(func() { virtualMachine = SMBIOSIsVirtualMachine(smbiosTable()) })
	return virtualMachine
}
