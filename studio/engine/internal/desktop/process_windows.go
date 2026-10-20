//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

var (
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

const swShowNormal = 1

// newProcessGroup is subprocess.CREATE_NEW_PROCESS_GROUP, so a Ctrl+C in the launching
// terminal does not reach the browser.
func newProcessGroup() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// openDefault is webbrowser.open on Windows, i.e. os.startfile(url):
// ShellExecuteW(NULL, NULL, url, NULL, NULL, SW_SHOWNORMAL). The default verb of a URL is
// "open". Failures are ignored, as webbrowser.open only returned False.
func openDefault(url string) {
	target, err := syscall.UTF16PtrFromString(url)
	if err != nil {
		return
	}
	_, _, _ = procShellExecuteW.Call(0, 0, uintptr(unsafe.Pointer(target)), 0, 0, swShowNormal)
}

// deleteFile is os.unlink on Windows: DeleteFileW only. os.Remove would also remove an
// empty directory and clear a read-only attribute, which Python's unlink does not.
func deleteFile(path string) error {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return syscall.DeleteFile(name)
}
