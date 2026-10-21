//go:build windows

package desktop

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")

	procAttachConsole = kernel32.NewProc("AttachConsole")
	procMessageBoxW   = user32.NewProc("MessageBoxW")
)

const (
	attachParentProcess = ^uintptr(0) // ATTACH_PARENT_PROCESS (DWORD)-1
	mbIconError         = 0x10
	mbIconInformation   = 0x40
)

// lineEnd is what Python's text streams write for "\n" on Windows.
const lineEnd = "\r\n"

// stdUsable mirrors CPython's is_valid_fd on Windows: a handle exists and GetFileType
// does not report FILE_TYPE_UNKNOWN. A windowed process started without a terminal has
// NULL standard handles, so Go's os.Stdout/os.Stderr wrap handle 0 and fail here.
func stdUsable(file *os.File) bool {
	if file == nil {
		return false
	}
	handle := syscall.Handle(file.Fd())
	if handle == 0 || handle == syscall.InvalidHandle {
		return false
	}
	kind, _ := syscall.GetFileType(handle)
	return kind != 0
}

// AttachParentTerminal is windowed.attach_parent_terminal: when headless, attach to the
// console of the process that started this one (AttachConsole(ATTACH_PARENT_PROCESS)) and
// point each missing os.Stdout/os.Stderr at CONOUT$, so `ScadaBench.exe --self-test` from
// a terminal prints there. Streams that were already usable are kept.
//
// Afterwards os.Stdout and os.Stderr are never nil: with no terminal they stay Go's
// default *os.File for the NULL standard handle (Fd() == 0; writes return an error).
// Passed to exec.Cmd such a file gives the child no handle for that stream, and Headless
// keeps returning true. Only if Go saw INVALID_HANDLE_VALUE (os.Stdout == nil) is the nil
// replaced by that same handle-0 file, because a nil *os.File in an io.Writer would make
// exec.Cmd duplicate an invalid handle.
func AttachParentTerminal() {
	if Headless() {
		attachConsole()
	}
	if os.Stdout == nil {
		os.Stdout = os.NewFile(0, "/dev/stdout")
	}
	if os.Stderr == nil {
		os.Stderr = os.NewFile(0, "/dev/stderr")
	}
}

func attachConsole() {
	if attached, _, _ := procAttachConsole.Call(attachParentProcess); attached == 0 {
		return
	}
	console, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	used := false
	if !stdUsable(os.Stdout) {
		os.Stdout, used = console, true
	}
	if !stdUsable(os.Stderr) {
		os.Stderr, used = console, true
	}
	if !used {
		_ = console.Close()
	}
}

// messageBox shows MessageBoxW(NULL, text, "ScadaBench", icon) and waits for OK.
func messageBox(text string, isError bool) {
	icon := uintptr(mbIconInformation)
	if isError {
		icon = mbIconError
	}
	body, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	caption, _ := syscall.UTF16PtrFromString(Title)
	_, _, _ = procMessageBoxW.Call(0, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(caption)), icon)
}
