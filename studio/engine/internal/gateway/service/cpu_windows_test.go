package service

import (
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	processCycles    = kernel32.NewProc("QueryProcessCycleTime")
	threadCycles     = kernel32.NewProc("QueryThreadCycleTime")
	cycleRateOnce    sync.Once
	cyclesPerSecond  float64
	currentThreadPsd = ^uintptr(1) // GetCurrentThread() pseudo handle (-2)
)

// processTicks is the user + kernel time of this process (GetProcessTimes). Windows charges
// it per 15.6 ms scheduler tick, so loads of a few milliseconds per second are coarse.
func processTicks() time.Duration {
	handle, err := syscall.GetCurrentProcess()
	if err != nil {
		return 0
	}
	var creation, exit, kernel, user syscall.Filetime
	if syscall.GetProcessTimes(handle, &creation, &exit, &kernel, &user) != nil {
		return 0
	}
	total := (int64(kernel.HighDateTime)<<32 | int64(kernel.LowDateTime)) +
		(int64(user.HighDateTime)<<32 | int64(user.LowDateTime))
	return time.Duration(total * 100)
}

// processCPU is the CPU time of this process from its exact cycle count
// (QueryProcessCycleTime), converted with a cycle rate measured on a spinning thread.
func processCPU() time.Duration {
	cycleRateOnce.Do(calibrate)
	handle, err := syscall.GetCurrentProcess()
	if err != nil || cyclesPerSecond == 0 {
		return processTicks()
	}
	var cycles uint64
	if ok, _, _ := processCycles.Call(uintptr(handle), uintptr(unsafe.Pointer(&cycles))); ok == 0 {
		return processTicks()
	}
	return time.Duration(float64(cycles) / cyclesPerSecond * float64(time.Second))
}

// calibrate spins one locked thread for 300 ms and divides its cycles by the elapsed time.
func calibrate() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var before, after uint64
	if ok, _, _ := threadCycles.Call(currentThreadPsd, uintptr(unsafe.Pointer(&before))); ok == 0 {
		return
	}
	start := time.Now()
	for time.Since(start) < 300*time.Millisecond {
	}
	elapsed := time.Since(start)
	if ok, _, _ := threadCycles.Call(currentThreadPsd, uintptr(unsafe.Pointer(&after))); ok == 0 {
		return
	}
	cyclesPerSecond = float64(after-before) / elapsed.Seconds()
}
