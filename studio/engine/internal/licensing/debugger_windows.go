//go:build windows

package licensing

import "unsafe"

var (
	procIsDebuggerPresent          = kernel32.NewProc("IsDebuggerPresent")
	procCheckRemoteDebuggerPresent = kernel32.NewProc("CheckRemoteDebuggerPresent")
	procGetCurrentProcess          = kernel32.NewProc("GetCurrentProcess")
)

// DebuggerAttached is integrity.debugger_attached: a local or remote debugger (read-only).
func DebuggerAttached() bool {
	local, _, _ := procIsDebuggerPresent.Call()
	process, _, _ := procGetCurrentProcess.Call()
	var remote int32
	procCheckRemoteDebuggerPresent.Call(process, uintptr(unsafe.Pointer(&remote)))
	return local != 0 || remote != 0
}
