//go:build windows

package netcfg

import (
	"strconv"
	"syscall"
	"unsafe"

	"studio-engine/internal/apperr"
)

// The UAC prompt (elevation.run_elevated). Only the confirmed apply/clean-up flow calls this;
// no test ever does.
var (
	shell32                 = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteExW     = shell32.NewProc("ShellExecuteExW")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procWaitForSingleObject = kernel32.NewProc("WaitForSingleObject")
	procGetExitCodeProcess  = kernel32.NewProc("GetExitCodeProcess")
)

const (
	seeMaskNoCloseProcess = 0x00000040
	swHide                = 0
	errorCancelled        = 1223
)

// shellExecuteInfo is SHELLEXECUTEINFOW.
type shellExecuteInfo struct {
	cbSize         uint32
	fMask          uint32
	hwnd           uintptr
	lpVerb         *uint16
	lpFile         *uint16
	lpParameters   *uint16
	lpDirectory    *uint16
	nShow          int32
	hInstApp       uintptr
	lpIDList       uintptr
	lpClass        *uint16
	hkeyClass      uintptr
	dwHotKey       uint32
	hIconOrMonitor uintptr
	hProcess       syscall.Handle
}

// UACElevator runs a program through ShellExecuteExW with the "runas" verb, hidden, and
// waits up to two minutes for it.
type UACElevator struct{}

// Elevate shows the administrator prompt, starts the program and returns its exit code.
func (UACElevator) Elevate(executable string, arguments []string) (uint32, error) {
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, err := syscall.UTF16PtrFromString(executable)
	if err != nil {
		return 0, err
	}
	parameters, err := syscall.UTF16PtrFromString(ListToCmdline(arguments))
	if err != nil {
		return 0, err
	}
	info := shellExecuteInfo{fMask: seeMaskNoCloseProcess, lpVerb: verb, lpFile: file,
		lpParameters: parameters, nShow: swHide}
	info.cbSize = uint32(unsafe.Sizeof(info))
	ok, _, callErr := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		code := uint32(0)
		if errno, isErrno := callErr.(syscall.Errno); isErrno {
			code = uint32(errno)
		}
		if code == errorCancelled {
			return 0, ErrElevationCancelled
		}
		return 0, apperr.RuntimeError("Windows could not start the network helper (error " +
			strconv.FormatUint(uint64(code), 10) + ")")
	}
	defer syscall.CloseHandle(info.hProcess)
	wait, _, _ := procWaitForSingleObject.Call(uintptr(info.hProcess), uintptr(HelperTimeout.Milliseconds()))
	if wait != 0 {
		return 0, apperr.RuntimeError("The network helper did not finish within two minutes")
	}
	var code uint32
	procGetExitCodeProcess.Call(uintptr(info.hProcess), uintptr(unsafe.Pointer(&code)))
	return code, nil
}
