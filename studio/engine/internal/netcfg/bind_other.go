//go:build !windows

package netcfg

import (
	"net"
	"strconv"
)

// SystemBinder binds a TCP listener and closes it again (development hosts off Windows).
type SystemBinder struct{}

// Bind reports whether address:port is free for a listener.
func (SystemBinder) Bind(address string, port int) error {
	listener, err := net.Listen("tcp4", net.JoinHostPort(address, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	return listener.Close()
}
