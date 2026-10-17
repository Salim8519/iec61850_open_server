package main

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// shell32 and ole32 are Windows KnownDLLs, always loaded from System32.
var (
	getKnownFolderPath = syscall.NewLazyDLL("shell32.dll").NewProc("SHGetKnownFolderPath")
	coTaskMemFree      = syscall.NewLazyDLL("ole32.dll").NewProc("CoTaskMemFree")
	folderDesktop      = syscall.GUID{Data1: 0xB4BFCC3A, Data2: 0xDB2C, Data3: 0x424C,
		Data4: [8]byte{0xB0, 0x29, 0x7F, 0xE9, 0x9A, 0x87, 0xC6, 0x41}}
)

// desktopFolder returns the real Desktop, also when OneDrive redirects it.
func desktopFolder() (string, error) {
	var path *uint16
	result, _, _ := getKnownFolderPath.Call(uintptr(unsafe.Pointer(&folderDesktop)), 0, 0,
		uintptr(unsafe.Pointer(&path)))
	if result != 0 || path == nil {
		home, err := os.UserHomeDir()
		return filepath.Join(home, "Desktop"), err
	}
	defer coTaskMemFree.Call(uintptr(unsafe.Pointer(path)))
	var chars []uint16
	for p := unsafe.Pointer(path); *(*uint16)(p) != 0; p = unsafe.Add(p, 2) {
		chars = append(chars, *(*uint16)(p))
	}
	return syscall.UTF16ToString(chars), nil
}
