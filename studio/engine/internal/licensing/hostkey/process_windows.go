//go:build windows

package hostkey

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// systemInstance is the single-instance mutex and the way to start the helper on Windows.
type systemInstance struct{}

// SystemProcess is the real process seam.
func SystemProcess() Process { return systemInstance{} }

func mutexName() *uint16 {
	name, _ := windows.UTF16PtrFromString(MutexName)
	return name
}

// Running reports whether a helper of this Windows session holds the mutex.
func (systemInstance) Running() bool {
	handle, err := windows.OpenMutex(windows.SYNCHRONIZE, false, mutexName())
	if err != nil {
		return false
	}
	windows.CloseHandle(handle)
	return true
}

// Start runs the helper without a window, detached from the app that started it.
func (systemInstance) Start(executable string) error {
	command := exec.Command(executable)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true,
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW}
	if err := command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}

// Acquire takes the mutex; ok is false when another helper already serves this session. release
// gives it back (the process also gives it back when it exits).
func Acquire() (release func(), ok bool) {
	handle, err := windows.CreateMutex(nil, true, mutexName())
	if err != nil { // ERROR_ALREADY_EXISTS comes back as an error together with a valid handle
		if handle != 0 {
			windows.CloseHandle(handle)
		}
		return nil, false
	}
	return func() { windows.CloseHandle(handle) }, true
}
