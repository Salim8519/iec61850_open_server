//go:build windows

package netcfg

import "unsafe"

var procSendARP = iphlpapi.NewProc("SendARP")

// sendARP is arp_probe._answers: true when a device answers an ARP request for address.
// It only asks; nothing on the PC changes.
func sendARP(address string) bool {
	ip, ok := parseIPv4(address)
	if !ok {
		return false
	}
	destination := uint32(ip>>24) | uint32(ip>>16&0xFF)<<8 | uint32(ip>>8&0xFF)<<16 | uint32(ip&0xFF)<<24
	var mac [8]byte
	length := uint32(8)
	result, _, _ := procSendARP.Call(uintptr(destination), 0, uintptr(unsafe.Pointer(&mac[0])),
		uintptr(unsafe.Pointer(&length)))
	return result == 0 && length > 0
}
