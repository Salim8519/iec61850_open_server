package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

// Only the standard syscall package is used (no golang.org/x/sys), so the helper
// links no networking code at all. kernel32 is a Windows KnownDLL: it always loads
// from System32, never from the folder (USB stick) the helper runs in.
var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	getFirmwareTable   = kernel32.NewProc("GetSystemFirmwareTable")
	getVolumeInfo      = kernel32.NewProc("GetVolumeInformationW")
	keyWow6464Key      = uint32(0x0100)
	cryptographyKey, _ = syscall.UTF16PtrFromString(`SOFTWARE\Microsoft\Cryptography`)
	machineGUIDName, _ = syscall.UTF16PtrFromString("MachineGuid")
)

// machineID reads the same three identifiers as the Studio app and native workers.
func machineID() (string, error) {
	guid, err := machineGUID()
	if err != nil {
		return "", fmt.Errorf("cannot read this computer's machine ID: %w", err)
	}
	serial, err := volumeSerial()
	if err != nil {
		return "", err
	}
	return composeHWID(guid, serial, firmwareUUID()), nil
}

func machineGUID() (string, error) {
	var key syscall.Handle
	err := syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, cryptographyKey, 0,
		syscall.KEY_QUERY_VALUE|keyWow6464Key, &key)
	if err != nil {
		return "", err
	}
	defer syscall.RegCloseKey(key)
	buffer := make([]uint16, 128)
	size := uint32(len(buffer) * 2)
	var kind uint32
	err = syscall.RegQueryValueEx(key, machineGUIDName, nil, &kind, (*byte)(unsafe.Pointer(&buffer[0])), &size)
	if err != nil {
		return "", err
	}
	if kind != syscall.REG_SZ {
		return "", fmt.Errorf("MachineGuid has an unexpected type")
	}
	return syscall.UTF16ToString(buffer[:size/2]), nil
}

func volumeSerial() (uint32, error) {
	drive := os.Getenv("SystemDrive")
	if drive == "" {
		drive = "C:"
	}
	root := strings.TrimRight(drive, `\`) + `\`
	path, _ := syscall.UTF16PtrFromString(root)
	var serial uint32
	ok, _, callErr := getVolumeInfo.Call(uintptr(unsafe.Pointer(path)), 0, 0,
		uintptr(unsafe.Pointer(&serial)), 0, 0, 0, 0)
	if ok == 0 {
		return 0, fmt.Errorf("cannot read the volume serial number of %s: %v", root, callErr)
	}
	return serial, nil
}

func firmwareUUID() string {
	const rsmb = 0x52534D42 // 'RSMB' raw SMBIOS firmware table provider
	size, _, _ := getFirmwareTable.Call(rsmb, 0, 0, 0)
	if size == 0 {
		return ""
	}
	buffer := make([]byte, size)
	if got, _, _ := getFirmwareTable.Call(rsmb, 0, uintptr(unsafe.Pointer(&buffer[0])), size); got == 0 {
		return ""
	}
	return smbiosSystemUUID(buffer)
}
