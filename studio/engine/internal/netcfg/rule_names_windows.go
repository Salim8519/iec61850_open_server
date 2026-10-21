//go:build windows

package netcfg

import (
	"syscall"
	"unsafe"
)

// The local firewall policy store; only read here, never written.
const firewallRuleStore = `SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\FirewallRules`

var (
	advapi32         = syscall.NewLazyDLL("advapi32.dll")
	procRegEnumValue = advapi32.NewProc("RegEnumValueW")
)

const (
	errorNoMoreItems = 259
	errorMoreData    = 234
)

// localRuleNames lists the value names of the firewall rule store, one per rule.
func localRuleNames() ([]string, error) {
	subkey, _ := syscall.UTF16PtrFromString(firewallRuleStore)
	var key syscall.Handle
	if err := syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, subkey, 0, syscall.KEY_READ|0x0100, &key); err != nil {
		return nil, err
	}
	defer syscall.RegCloseKey(key)
	names := []string{}
	buffer := make([]uint16, 512)
	for index := uint32(0); ; {
		size := uint32(len(buffer))
		result, _, _ := procRegEnumValue.Call(uintptr(key), uintptr(index), uintptr(unsafe.Pointer(&buffer[0])),
			uintptr(unsafe.Pointer(&size)), 0, 0, 0, 0)
		switch result {
		case 0:
			names = append(names, syscall.UTF16ToString(buffer[:size]))
			index++
		case errorNoMoreItems:
			return names, nil
		case errorMoreData:
			if len(buffer) >= 32768 {
				return nil, syscall.Errno(result)
			}
			buffer = make([]uint16, len(buffer)*4)
		default:
			return nil, syscall.Errno(result)
		}
	}
}
