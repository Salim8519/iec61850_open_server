//go:build !windows

package runtime

import (
	"context"
	"syscall"
)

// exclusiveAddress has no SO_EXCLUSIVEADDRUSE outside Windows; the product ships for Windows.
func exclusiveAddress(network, address string, conn syscall.RawConn) error { return nil }

func contextBackground() context.Context { return context.Background() }
