package main

import (
	"os"
	"path/filepath"
	"syscall"
)

// millisecondTimer asks Windows for 1 ms timer resolution while the engine runs,
// like studio_ied does (change 0102), so the uplink's 1 ms waits stay 1 ms.
func millisecondTimer() func() {
	winmm, err := syscall.LoadDLL(filepath.Join(os.Getenv("SystemRoot"), "System32", "winmm.dll"))
	if err != nil {
		return func() {}
	}
	begin, errBegin := winmm.FindProc("timeBeginPeriod")
	end, errEnd := winmm.FindProc("timeEndPeriod")
	if errBegin != nil || errEnd != nil {
		return func() {}
	}
	begin.Call(1)
	return func() { end.Call(1) }
}
