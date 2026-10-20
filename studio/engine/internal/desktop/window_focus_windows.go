//go:build windows

package desktop

import (
	"sync"
	"syscall"
	"unsafe"
)

var (
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetWindow                = user32.NewProc("GetWindow")
	procIsIconic                 = user32.NewProc("IsIconic")
	procShowWindow               = user32.NewProc("ShowWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
)

const (
	gwOwner   = 4 // GW_OWNER: skip pop-ups owned by the main window
	swRestore = 9
)

// One EnumWindows callback for the process (syscall.NewCallback slots are never freed);
// enumMu serialises searches, which share enumPID and enumFound.
var (
	enumOnce     sync.Once
	enumCallback uintptr
	enumMu       sync.Mutex
	enumPID      uint32
	enumFound    uintptr
)

func collectWindow(window, _ uintptr) uintptr {
	var owner uint32
	_, _, _ = procGetWindowThreadProcessId.Call(window, uintptr(unsafe.Pointer(&owner)))
	if owner != enumPID || enumFound != 0 {
		return 1
	}
	visible, _, _ := procIsWindowVisible.Call(window)
	ownerWindow, _, _ := procGetWindow.Call(window, gwOwner)
	if visible != 0 && ownerWindow == 0 {
		enumFound = window
	}
	return 1
}

// firstVisibleWindow is window_focus._visible_windows(pid)[0]: the first visible,
// unowned top-level window of pid in EnumWindows order, or 0.
func firstVisibleWindow(pid int) uintptr {
	enumOnce.Do(func() { enumCallback = syscall.NewCallback(collectWindow) })
	enumMu.Lock()
	defer enumMu.Unlock()
	enumPID, enumFound = uint32(pid), 0
	_, _, _ = procEnumWindows.Call(enumCallback, 0)
	return enumFound
}

// FocusProcessWindow is window_focus.focus_process_window: bring the first visible,
// unowned top-level window of process pid to the front, restoring it when minimised.
// True when such a window existed (even if Windows refused the foreground change).
func FocusProcessWindow(pid int) bool {
	if pid == 0 {
		return false
	}
	window := firstVisibleWindow(pid)
	if window == 0 {
		return false
	}
	if iconic, _, _ := procIsIconic.Call(window); iconic != 0 {
		_, _, _ = procShowWindow.Call(window, swRestore)
	}
	_, _, _ = procSetForegroundWindow.Call(window)
	return true
}

// AllowForeground calls AllowSetForegroundWindow(pid) so process pid (for example the
// staged ScadaBench.exe the portable launcher just started) may bring a window to the
// front. Errors are ignored.
func AllowForeground(pid int) {
	_, _, _ = procAllowSetForegroundWindow.Call(uintptr(uint32(pid)))
}
