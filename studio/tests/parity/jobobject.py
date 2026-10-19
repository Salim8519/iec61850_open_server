"""Windows Job object that kills an engine and every worker it started when closed.

The harness puts each spawned engine into its own job (KILL_ON_JOB_CLOSE). After the
engine's own shutdown it lists the processes still in the job: a surviving studio_*.exe
is an orphan worker, which the user would notice, so the test fails.
"""
import ctypes
from ctypes import wintypes
from pathlib import Path

KILL_ON_JOB_CLOSE = 0x2000
EXTENDED_LIMIT_INFORMATION = 9
BASIC_PROCESS_ID_LIST = 3
QUERY_LIMITED_INFORMATION = 0x1000


class _Basic(ctypes.Structure):
    _fields_ = [("PerProcessUserTimeLimit", ctypes.c_int64), ("PerJobUserTimeLimit", ctypes.c_int64),
                ("LimitFlags", wintypes.DWORD), ("MinimumWorkingSetSize", ctypes.c_size_t),
                ("MaximumWorkingSetSize", ctypes.c_size_t), ("ActiveProcessLimit", wintypes.DWORD),
                ("Affinity", ctypes.c_size_t), ("PriorityClass", wintypes.DWORD),
                ("SchedulingClass", wintypes.DWORD)]


class _Extended(ctypes.Structure):
    _fields_ = [("BasicLimitInformation", _Basic), ("IoInfo", ctypes.c_uint64 * 6),
                ("ProcessMemoryLimit", ctypes.c_size_t), ("JobMemoryLimit", ctypes.c_size_t),
                ("PeakProcessMemoryUsed", ctypes.c_size_t), ("PeakJobMemoryUsed", ctypes.c_size_t)]


def _kernel():
    kernel = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel.CreateJobObjectW.restype = wintypes.HANDLE
    kernel.CreateJobObjectW.argtypes = [wintypes.LPVOID, wintypes.LPCWSTR]
    kernel.SetInformationJobObject.argtypes = [wintypes.HANDLE, ctypes.c_int, wintypes.LPVOID, wintypes.DWORD]
    kernel.AssignProcessToJobObject.argtypes = [wintypes.HANDLE, wintypes.HANDLE]
    kernel.QueryInformationJobObject.argtypes = [wintypes.HANDLE, ctypes.c_int, wintypes.LPVOID,
                                                 wintypes.DWORD, wintypes.LPVOID]
    kernel.OpenProcess.restype = wintypes.HANDLE
    kernel.OpenProcess.argtypes = [wintypes.DWORD, wintypes.BOOL, wintypes.DWORD]
    kernel.QueryFullProcessImageNameW.argtypes = [wintypes.HANDLE, wintypes.DWORD, wintypes.LPWSTR,
                                                  ctypes.POINTER(wintypes.DWORD)]
    kernel.CloseHandle.argtypes = [wintypes.HANDLE]
    return kernel


class Job:
    def __init__(self):
        self._kernel = _kernel()
        self.handle = self._kernel.CreateJobObjectW(None, None)
        if not self.handle:
            raise ctypes.WinError(ctypes.get_last_error())
        info = _Extended()
        info.BasicLimitInformation.LimitFlags = KILL_ON_JOB_CLOSE
        if not self._kernel.SetInformationJobObject(self.handle, EXTENDED_LIMIT_INFORMATION,
                                                    ctypes.byref(info), ctypes.sizeof(info)):
            raise ctypes.WinError(ctypes.get_last_error())

    def assign(self, popen):
        if not self._kernel.AssignProcessToJobObject(self.handle, wintypes.HANDLE(int(popen._handle))):
            raise ctypes.WinError(ctypes.get_last_error())

    def pids(self):
        if not self.handle:
            return []
        count = 1024

        class _List(ctypes.Structure):
            _fields_ = [("Assigned", wintypes.DWORD), ("Listed", wintypes.DWORD),
                        ("Ids", ctypes.c_size_t * count)]
        listing = _List()
        if not self._kernel.QueryInformationJobObject(self.handle, BASIC_PROCESS_ID_LIST, ctypes.byref(listing),
                                                      ctypes.sizeof(listing), None):
            raise ctypes.WinError(ctypes.get_last_error())
        return [int(listing.Ids[index]) for index in range(listing.Listed)]

    def processes(self):
        """{pid: image file name} of every process still in the job."""
        return {pid: image_name(pid, self._kernel) for pid in self.pids()}

    def close(self):
        if self.handle:
            self._kernel.CloseHandle(self.handle)
            self.handle = None


def image_name(pid, kernel=None):
    kernel = kernel or _kernel()
    handle = kernel.OpenProcess(QUERY_LIMITED_INFORMATION, False, pid)
    if not handle:
        return ""
    try:
        size = wintypes.DWORD(1024)
        buffer = ctypes.create_unicode_buffer(size.value)
        if not kernel.QueryFullProcessImageNameW(handle, 0, buffer, ctypes.byref(size)):
            return ""
        return Path(buffer.value).name
    finally:
        kernel.CloseHandle(handle)
