//go:build !windows

package licensing

// IsVirtualMachine is false outside Windows: there is no firmware table to read, and a computer that
// cannot tell never uses a host's license.
func IsVirtualMachine() bool { return false }
