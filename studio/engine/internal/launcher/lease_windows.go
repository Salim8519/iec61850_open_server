//go:build windows

package launcher

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

const (
	lockfileFailImmediately = 0x1
	lockfileExclusiveLock   = 0x2
)

// Lease holds the data folder's studio.lock byte lock for the life of the process.
type Lease struct{ file *os.File }

// AcquireLease is launcher/directory_lease.py DirectoryLease.__enter__: create the folder,
// refuse a linked lock file, write "0" into an empty lock file, then lock byte 0
// exclusively without waiting. Python's msvcrt.locking(LK_NBLCK, 1) takes the same
// Win32 lock (LockFile on [0, 1)), so a Python and a Go engine exclude each other.
func AcquireLease(directory string) (*Lease, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(directory, "studio.lock")
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New(LeaseLinkText)
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, errors.New(LeaseBusyText)
	}
	if info, err := file.Stat(); err == nil && info.Size() == 0 {
		_, _ = file.Write([]byte("0"))
	}
	var overlapped syscall.Overlapped
	result, _, _ := procLockFileEx.Call(file.Fd(), lockfileExclusiveLock|lockfileFailImmediately, 0, 1, 0,
		uintptr(unsafe.Pointer(&overlapped)))
	if result == 0 {
		_ = file.Close()
		return nil, errors.New(LeaseBusyText)
	}
	return &Lease{file: file}, nil
}

// Release unlocks byte 0 and closes the lock file (DirectoryLease.__exit__).
func (l *Lease) Release() {
	if l == nil || l.file == nil {
		return
	}
	var overlapped syscall.Overlapped
	_, _, _ = procUnlockFileEx.Call(l.file.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
	_ = l.file.Close()
	l.file = nil
}
