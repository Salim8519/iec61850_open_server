//go:build !windows

package service

import (
	"syscall"
	"time"
)

// processCPU is the user + system time of this process (getrusage).
func processCPU() time.Duration {
	var usage syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &usage) != nil {
		return 0
	}
	return time.Duration(usage.Utime.Nano() + usage.Stime.Nano())
}

// processTicks is processCPU: getrusage has no coarser tick-based variant to compare.
func processTicks() time.Duration {
	return processCPU()
}
