//go:build windows

package netcfg

import (
	"fmt"
	"unsafe"
)

// HasAddress says whether any card of this PC has the IPv4 address now, in any state. It reads
// Windows' unicast address table once: no adapter listing and no DNS Client work, so a running
// gateway can ask every few seconds (change 0221, trial N8).
func HasAddress(ip string) (bool, error) {
	var table unsafe.Pointer
	result, _, _ := procGetUnicastIpAddressTable.Call(afInet, uintptr(unsafe.Pointer(&table)))
	if result != 0 {
		return false, fmt.Errorf("[WinError %d] GetUnicastIpAddressTable failed", result)
	}
	defer procFreeMibTable.Call(uintptr(table))
	count := *(*uint32)(table)
	for row := 0; row < int(count); row++ {
		if sockaddrIPv4((*[8]byte)(unsafe.Add(table, 8+row*80))) == ip { // 80-byte rows on x64
			return true, nil
		}
	}
	return false, nil
}
