//go:build windows

package worker

import (
	"sync"
	"syscall"
	"unsafe"
)

// Tool workers (studio_live, studio_control, studio_gateway, the master) join one Job Object
// with JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE, as internal/native does for simulation workers, so
// no studio_* child outlives the server even when the server is killed hard.
const (
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

// newKillOnCloseJob creates an unnamed, non-inheritable job that kills its members when its
// last handle closes.
func newKillOnCloseJob() (syscall.Handle, error) {
	handle, _, callErr := procCreateJobObject.Call(0, 0)
	if handle == 0 {
		return 0, callErr
	}
	var info extendedLimitInformation
	info.BasicLimitInformation.LimitFlags = jobLimitKillOnJobClose
	ok, _, callErr := procSetInformationJob.Call(handle, jobObjectExtendedLimit,
		uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info))
	if ok == 0 {
		syscall.CloseHandle(syscall.Handle(handle))
		return 0, callErr
	}
	return syscall.Handle(handle), nil
}

// serverJob is created once; its handle stays open for the life of the server and Windows
// closes it (killing every tool worker) when the server exits for any reason.
func serverJob() syscall.Handle {
	jobOnce.Do(func() { jobHandle, _ = newKillOnCloseJob() })
	return jobHandle
}

// assign puts process pid into job.
func assign(job syscall.Handle, pid int) error {
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

// adopt puts a started worker into the server's job. A failure leaves the worker running
// exactly as Python starts it (no job); Close still stops it.
func adopt(pid int) error { return assign(serverJob(), pid) }
