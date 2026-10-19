//go:build windows

package runtime

import (
	"context"
	"syscall"
)

// soExclusiveAddrUse is SO_EXCLUSIVEADDRUSE (~SO_REUSEADDR).
const soExclusiveAddrUse = ^4

func exclusiveAddress(network, address string, conn syscall.RawConn) error {
	var failure error
	err := conn.Control(func(fd uintptr) {
		failure = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, soExclusiveAddrUse, 1)
	})
	if err != nil {
		return err
	}
	return failure
}

func contextBackground() context.Context { return context.Background() }
