//go:build windows

package licensing

import (
	"syscall"
	"unsafe"
)

// WindowsRegistry is clock_store.WindowsRegistry: string values under
// HKEY_CURRENT_USER\<Key> (per user, no administrator rights). Key "" means RegistryKey.
type WindowsRegistry struct{ Key string }

const (
	regSZ        = 1
	keySetValue  = 0x0002
	regOptionNon = 0
)

var (
	advapi32           = syscall.NewLazyDLL("advapi32.dll")
	procRegCreateKeyEx = advapi32.NewProc("RegCreateKeyExW")
	procRegSetValueEx  = advapi32.NewProc("RegSetValueExW")
)

func (r WindowsRegistry) key() string {
	if r.Key == "" {
		return RegistryKey
	}
	return r.Key
}

// Get reads a REG_SZ value; anything else, or any failure, is "not there".
func (r WindowsRegistry) Get(name string) (string, bool) {
	subkey, err := syscall.UTF16PtrFromString(r.key())
	if err != nil {
		return "", false
	}
	var key syscall.Handle
	if syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, subkey, 0, syscall.KEY_READ, &key) != nil {
		return "", false
	}
	defer syscall.RegCloseKey(key)
	value, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return "", false
	}
	var kind, size uint32
	if syscall.RegQueryValueEx(key, value, nil, &kind, nil, &size) != nil || kind != regSZ {
		return "", false
	}
	buffer := make([]uint16, size/2+1)
	if syscall.RegQueryValueEx(key, value, nil, &kind, (*byte)(unsafe.Pointer(&buffer[0])), &size) != nil {
		return "", false
	}
	return syscall.UTF16ToString(buffer), true
}

// Set writes a REG_SZ value, creating the key; every failure is ignored.
func (r WindowsRegistry) Set(name, text string) {
	subkey, err1 := syscall.UTF16PtrFromString(r.key())
	value, err2 := syscall.UTF16FromString(text)
	valueName, err3 := syscall.UTF16PtrFromString(name)
	if err1 != nil || err2 != nil || err3 != nil {
		return
	}
	var key syscall.Handle
	status, _, _ := procRegCreateKeyEx.Call(uintptr(syscall.HKEY_CURRENT_USER), uintptr(unsafe.Pointer(subkey)), 0, 0,
		regOptionNon, keySetValue, 0, uintptr(unsafe.Pointer(&key)), 0)
	if status != 0 {
		return
	}
	defer syscall.RegCloseKey(key)
	procRegSetValueEx.Call(uintptr(key), uintptr(unsafe.Pointer(valueName)), 0, regSZ,
		uintptr(unsafe.Pointer(&value[0])), uintptr(len(value)*2))
}
