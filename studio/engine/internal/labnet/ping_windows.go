//go:build windows

package labnet

import (
	"net/netip"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procIcmpCreateFile  = iphlpapi.NewProc("IcmpCreateFile")
	procIcmpCloseHandle = iphlpapi.NewProc("IcmpCloseHandle")
	procIcmpSendEcho    = iphlpapi.NewProc("IcmpSendEcho")
)

// icmpEchoReply is ICMP_ECHO_REPLY on 64-bit Windows.
type icmpEchoReply struct {
	Address       uint32
	Status        uint32
	RoundTripTime uint32
	DataSize      uint16
	Reserved      uint16
	Data          uintptr
	Options       [16]byte // IP_OPTION_INFORMATION (TTL, TOS, flags, size, pointer)
}

// Ping sends one ICMP echo, as Windows ping does; no administrator rights are needed.
func Ping(address string, timeoutMS uint32) (Pinged, error) {
	parsed, err := netip.ParseAddr(address)
	if err != nil || !parsed.Is4() {
		return Pinged{}, ErrAddress
	}
	handle, _, callErr := procIcmpCreateFile.Call()
	if windows.Handle(handle) == windows.InvalidHandle {
		return Pinged{}, callErr
	}
	defer procIcmpCloseHandle.Call(handle)
	octets := parsed.As4()
	destination := uint32(octets[0]) | uint32(octets[1])<<8 | uint32(octets[2])<<16 | uint32(octets[3])<<24
	payload := []byte("ScadaBench lab ping 0123456789ab")
	reply := make([]byte, int(unsafe.Sizeof(icmpEchoReply{}))+len(payload)+8)
	count, _, _ := procIcmpSendEcho.Call(handle, uintptr(destination), uintptr(unsafe.Pointer(&payload[0])),
		uintptr(len(payload)), 0, uintptr(unsafe.Pointer(&reply[0])), uintptr(len(reply)), uintptr(timeoutMS))
	if count == 0 {
		return Pinged{}, nil
	}
	echo := (*icmpEchoReply)(unsafe.Pointer(&reply[0]))
	return Pinged{OK: echo.Status == 0, MS: echo.RoundTripTime, TTL: echo.Options[0]}, nil
}
