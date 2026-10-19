//go:build windows

package netcfg

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
)

const soExclusiveAddrUse = ^int(syscall.SO_REUSEADDR) // ~SO_REUSEADDR, as winsock defines it

// SystemBinder binds a TCP socket with SO_EXCLUSIVEADDRUSE and closes it again, like
// endpoints.preflight. It never listens and changes nothing.
type SystemBinder struct{}

// Bind reports whether address:port is free for an exclusive listener.
func (SystemBinder) Bind(address string, port int) error {
	ip, ok := parseIPv4(address)
	if !ok {
		return fmt.Errorf("illegal IP address string passed to inet_aton")
	}
	handle, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		return winError(err)
	}
	defer syscall.Closesocket(handle)
	if err := syscall.SetsockoptInt(handle, syscall.SOL_SOCKET, soExclusiveAddrUse, 1); err != nil {
		return winError(err)
	}
	sockaddr := &syscall.SockaddrInet4{Port: port, Addr: [4]byte{byte(ip >> 24), byte(ip >> 16), byte(ip >> 8), byte(ip)}}
	if err := syscall.Bind(handle, sockaddr); err != nil {
		return winError(err)
	}
	return nil
}

// winError formats an error like Python's OSError on Windows: "[WinError N] message" with the
// trailing period and spaces removed.
func winError(err error) error {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return err
	}
	return fmt.Errorf("[WinError %d] %s", uint32(errno), strings.TrimRight(errno.Error(), " .\r\n\t"))
}
