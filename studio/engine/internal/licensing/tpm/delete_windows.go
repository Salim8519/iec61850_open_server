//go:build windows

package tpm

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ncrypt          = windows.NewLazyDLL("ncrypt.dll")
	openProvider    = ncrypt.NewProc("NCryptOpenStorageProvider")
	openStoredKey   = ncrypt.NewProc("NCryptOpenKey")
	deleteStoredKey = ncrypt.NewProc("NCryptDeleteKey")
	freeObject      = ncrypt.NewProc("NCryptFreeObject")
)

// deleteKey removes the attestation key called name from the Platform Crypto Provider.
// Every failure is ignored: a key that cannot be deleted is only a small leftover file.
func deleteKey(name string) {
	provider, err := windows.UTF16PtrFromString("Microsoft Platform Crypto Provider")
	keyName, err2 := windows.UTF16PtrFromString(name)
	if err != nil || err2 != nil {
		return
	}
	var handle, key uintptr
	if status, _, _ := openProvider.Call(uintptr(unsafe.Pointer(&handle)), uintptr(unsafe.Pointer(provider)), 0); status != 0 {
		return
	}
	defer freeObject.Call(handle)
	if status, _, _ := openStoredKey.Call(handle, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(keyName)), 0, 0); status != 0 {
		return
	}
	deleteStoredKey.Call(key, 0) // frees the key handle as well
}
