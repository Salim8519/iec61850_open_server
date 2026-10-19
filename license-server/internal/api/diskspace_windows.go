//go:build windows

package api

import (
	"syscall"
	"unsafe"
)

var getDiskFreeSpace = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")

// diskSpace returns the size and free bytes of the drive holding path (local runs).
func diskSpace(path string) (int64, int64, error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	var available, total, free uint64
	result, _, callErr := getDiskFreeSpace.Call(uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(&available)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&free)))
	if result == 0 {
		return 0, 0, callErr
	}
	return int64(total), int64(available), nil
}
