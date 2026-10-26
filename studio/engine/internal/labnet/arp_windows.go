//go:build windows

package labnet

import (
	"net/netip"
	"unsafe"
)

var procSendARP = iphlpapi.NewProc("SendARP")

// AskOn asks the network of the real card whose own address is source whether a device answers
// at address. The source pins the question to that card: without it Windows answers from the
// lab card itself for an address the lab card already holds (measured, change 0192).
func AskOn(address, source string) bool {
	destination, ok := wire(address)
	from, fromOK := wire(source)
	if !ok || !fromOK {
		return false
	}
	var mac [8]byte
	length := uint32(len(mac))
	result, _, _ := procSendARP.Call(uintptr(destination), uintptr(from), uintptr(unsafe.Pointer(&mac[0])),
		uintptr(unsafe.Pointer(&length)))
	return result == 0 && length > 0
}

// wire is the address in network byte order as IPAddr expects.
func wire(text string) (uint32, bool) {
	address, err := netip.ParseAddr(text)
	if err != nil || !address.Is4() {
		return 0, false
	}
	octets := address.As4()
	return uint32(octets[0]) | uint32(octets[1])<<8 | uint32(octets[2])<<16 | uint32(octets[3])<<24, true
}

// InstalledCard is the card GUID the installer stored; any user may read it ("" when none).
func InstalledCard() string {
	card, _ := readCard()
	return card
}
