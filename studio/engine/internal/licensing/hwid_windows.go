//go:build windows

package licensing

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

// Read-only Windows calls behind the machine ID: the raw SMBIOS table, the MachineGuid
// registry value (only for PCs without firmware identity) and, for trial parts, the system
// drive's volume serial. Nothing is ever written.
const (
	keyWow6464Key = 0x0100
	rsmbProvider  = 0x52534D42 // 'RSMB' raw SMBIOS firmware table provider
)

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procVolumeInformation = kernel32.NewProc("GetVolumeInformationW")
	procFirmwareTable     = kernel32.NewProc("GetSystemFirmwareTable")
)

func readMachineParts() (MachineParts, error) {
	table := smbiosTable()
	parts := MachineParts{FirmwareUUID: SMBIOSSystemUUID(table), BoardSerial: SMBIOSBoardSerial(table)}
	if CleanFirmwareUUID(parts.FirmwareUUID) == "" && CleanBoardSerial(parts.BoardSerial) == "" {
		guid, err := machineGUID() // only PCs without firmware identity fall back to this
		if err != nil {
			return MachineParts{}, err
		}
		parts.MachineGUID = guid
	}
	return parts, nil
}

// machineGUID reads HKLM\SOFTWARE\Microsoft\Cryptography\MachineGuid from the 64-bit view.
func machineGUID() (string, error) {
	subkey, _ := syscall.UTF16PtrFromString(`SOFTWARE\Microsoft\Cryptography`)
	var key syscall.Handle
	if err := syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, subkey, 0,
		syscall.KEY_READ|keyWow6464Key, &key); err != nil {
		return "", err
	}
	defer syscall.RegCloseKey(key)
	name, _ := syscall.UTF16PtrFromString("MachineGuid")
	var kind, size uint32
	if err := syscall.RegQueryValueEx(key, name, nil, &kind, nil, &size); err != nil {
		return "", err
	}
	buffer := make([]uint16, size/2+1)
	if err := syscall.RegQueryValueEx(key, name, nil, &kind, (*byte)(unsafe.Pointer(&buffer[0])), &size); err != nil {
		return "", err
	}
	return syscall.UTF16ToString(buffer), nil
}

// volumeSerial is the serial number of %SystemDrive% (default C:).
func volumeSerial() (uint32, error) {
	drive := os.Getenv("SystemDrive")
	if drive == "" {
		drive = "C:"
	}
	root := strings.TrimRight(drive, `\`) + `\`
	path, _ := syscall.UTF16PtrFromString(root)
	var serial uint32
	ok, _, _ := procVolumeInformation.Call(uintptr(unsafe.Pointer(path)), 0, 0,
		uintptr(unsafe.Pointer(&serial)), 0, 0, 0, 0)
	if ok == 0 {
		return 0, fmt.Errorf("Cannot read the volume serial number of %s", root)
	}
	return serial, nil
}

// smbiosTable is the raw SMBIOS firmware table (RawSMBIOSData), or nil when unreadable.
func smbiosTable() []byte {
	if procFirmwareTable.Find() != nil {
		return nil
	}
	size, _, _ := procFirmwareTable.Call(rsmbProvider, 0, 0, 0)
	if size == 0 {
		return nil
	}
	buffer := make([]byte, size)
	if got, _, _ := procFirmwareTable.Call(rsmbProvider, 0, uintptr(unsafe.Pointer(&buffer[0])), size); got == 0 {
		return nil
	}
	return buffer
}

// firmwareUUID is the SMBIOS system UUID, or "" when the table cannot be read.
func firmwareUUID() string { return SMBIOSSystemUUID(smbiosTable()) }

// readRawTrialParts is what trial_parts.machine_parts reads; unreadable parts stay nil.
func readRawTrialParts() RawTrialParts {
	var raw RawTrialParts
	if guid, err := machineGUID(); err == nil {
		raw.MachineGUID = &guid
	}
	if serial, err := volumeSerial(); err == nil {
		raw.VolumeSerial = &serial
	}
	raw.FirmwareUUID = firmwareUUID()
	return raw
}
