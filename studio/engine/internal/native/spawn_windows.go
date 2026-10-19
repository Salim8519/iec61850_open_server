//go:build windows

package native

import (
	"os/exec"
	"sync"
	"syscall"
	"unsafe"
)

// Workers start without a console window (CREATE_NO_WINDOW, as worker_process.py) and inside
// one Job Object with JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE, so no worker outlives the server
// even when the server is killed (architecture §2).
const (
	createNoWindow         = 0x08000000
	jobObjectExtendedLimit = 9
	jobLimitKillOnJobClose = 0x2000
	processSetQuota        = 0x0100
	processTerminate       = 0x0001
)

var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObject    = kernel32.NewProc("CreateJobObjectW")
	procSetInformationJob  = kernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJob = kernel32.NewProc("AssignProcessToJobObject")
	procOpenProcess        = kernel32.NewProc("OpenProcess")
	jobOnce                sync.Once
	jobHandle              syscall.Handle
)

type ioCounters struct {
	ReadOperationCount, WriteOperationCount, OtherOperationCount uint64
	ReadTransferCount, WriteTransferCount, OtherTransferCount    uint64
}

type basicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type extendedLimitInformation struct {
	BasicLimitInformation basicLimitInformation
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

func prepareCommand(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}

// killOnClose creates the server's job once; the handle stays open for the life of the
// process, and Windows closes it (killing every worker) when the server exits.
func killOnClose() syscall.Handle {
	jobOnce.Do(func() {
		handle, _, _ := procCreateJobObject.Call(0, 0)
		if handle == 0 {
			return
		}
		var info extendedLimitInformation
		info.BasicLimitInformation.LimitFlags = jobLimitKillOnJobClose
		ok, _, _ := procSetInformationJob.Call(handle, jobObjectExtendedLimit,
			uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info))
		if ok == 0 {
			syscall.CloseHandle(syscall.Handle(handle))
			return
		}
		jobHandle = syscall.Handle(handle)
	})
	return jobHandle
}

// adopt puts a started worker into the server's job; failure leaves the worker running
// exactly as Python starts it (no job), which is reported by the caller's tests only.
func adopt(pid int) error {
	job := killOnClose()
	if job == 0 {
		return syscall.EINVAL
	}
	process, _, callErr := procOpenProcess.Call(processSetQuota|processTerminate, 0, uintptr(pid))
	if process == 0 {
		return callErr
	}
	defer syscall.CloseHandle(syscall.Handle(process))
	ok, _, callErr := procAssignProcessToJob.Call(uintptr(job), process)
	if ok == 0 {
		return callErr
	}
	return nil
}
