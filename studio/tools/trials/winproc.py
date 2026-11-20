"""Kernel process times: when a process was created and exited, and how much CPU it used.

`Tracker.hold(pid)` opens a handle as soon as a worker is seen, so its exact creation and exit
times stay readable after it exits. `cpu_seconds(pids)` reads user+kernel CPU from the system process
list (what Task Manager reads), so it works for the DNS Client's svchost without admin rights.
"""
import ctypes
from ctypes import wintypes
import re
import subprocess

from winclock import filetime_to_unix

QUERY_LIMITED_INFORMATION = 0x1000
SYNCHRONIZE = 0x100000
STILL_ACTIVE = 259

_kernel = ctypes.WinDLL("kernel32", use_last_error=True)
_kernel.OpenProcess.restype = wintypes.HANDLE
_kernel.OpenProcess.argtypes = [wintypes.DWORD, wintypes.BOOL, wintypes.DWORD]
_kernel.GetProcessTimes.argtypes = [wintypes.HANDLE] + [ctypes.POINTER(wintypes.FILETIME)] * 4
_kernel.GetExitCodeProcess.argtypes = [wintypes.HANDLE, ctypes.POINTER(wintypes.DWORD)]
_kernel.CloseHandle.argtypes = [wintypes.HANDLE]
_kernel.GetSystemTimes.argtypes = [ctypes.POINTER(wintypes.FILETIME)] * 3


def _ticks(stamp):
    return (stamp.dwHighDateTime << 32) | stamp.dwLowDateTime


def _times(handle):
    created, exited, kernel, user = (wintypes.FILETIME() for _ in range(4))
    if not _kernel.GetProcessTimes(handle, ctypes.byref(created), ctypes.byref(exited),
                                   ctypes.byref(kernel), ctypes.byref(user)):
        raise ctypes.WinError(ctypes.get_last_error())
    return _ticks(created), _ticks(exited), (_ticks(kernel) + _ticks(user)) / 1e7


def _open(pid, rights=QUERY_LIMITED_INFORMATION | SYNCHRONIZE):
    return _kernel.OpenProcess(rights, False, pid)


class Tracker:
    """Holds a handle per process so creation and exit times survive the process."""

    def __init__(self):
        self.handles = {}

    def hold(self, pid):
        if pid not in self.handles:
            handle = _open(pid)
            if handle:
                self.handles[pid] = handle

    def times(self, pid):
        """(created, exited) in Unix seconds; exited is None while the process runs."""
        handle = self.handles.get(pid)
        if not handle:
            return None, None
        created, exited, _ = _times(handle)
        code = wintypes.DWORD()
        _kernel.GetExitCodeProcess(handle, ctypes.byref(code))
        running = code.value == STILL_ACTIVE and not exited
        return filetime_to_unix(created), None if running else filetime_to_unix(exited)

    def close(self):
        for handle in self.handles.values():
            _kernel.CloseHandle(handle)
        self.handles.clear()


class _ProcessInfo(ctypes.Structure):  # head of SYSTEM_PROCESS_INFORMATION (x64)
    _fields_ = [("NextEntryOffset", wintypes.ULONG), ("NumberOfThreads", wintypes.ULONG),
                ("Reserved", ctypes.c_byte * 24), ("CreateTime", ctypes.c_int64), ("UserTime", ctypes.c_int64),
                ("KernelTime", ctypes.c_int64), ("ImageName", ctypes.c_byte * 16), ("BasePriority", ctypes.c_long),
                ("UniqueProcessId", ctypes.c_void_p)]


_ntdll = ctypes.WinDLL("ntdll")
_ntdll.NtQuerySystemInformation.argtypes = [ctypes.c_int, ctypes.c_void_p, wintypes.ULONG,
                                            ctypes.POINTER(wintypes.ULONG)]


def cpu_seconds(pids):
    """{pid: user + kernel CPU seconds so far} for the given pids. Uses the system process list
    (what Task Manager reads), so it also works for service processes a normal user cannot open."""
    size = 1 << 20
    while True:
        buffer = ctypes.create_string_buffer(size)
        needed = wintypes.ULONG()
        status = _ntdll.NtQuerySystemInformation(5, buffer, size, ctypes.byref(needed)) & 0xFFFFFFFF
        if status != 0xC0000004:  # STATUS_INFO_LENGTH_MISMATCH
            break
        size = needed.value + (1 << 16)
    if status:
        raise OSError(status, "NtQuerySystemInformation failed")
    wanted, found, offset = set(pids), {}, 0
    while True:
        info = _ProcessInfo.from_buffer_copy(buffer, offset)
        pid = info.UniqueProcessId or 0
        if pid in wanted:
            found[pid] = (info.UserTime + info.KernelTime) / 1e7
        if not info.NextEntryOffset:
            return found
        offset += info.NextEntryOffset


def system_busy():
    """(busy, total) CPU seconds of the whole machine so far, all processors (GetSystemTimes);
    the share busy between two readings is (busy2 - busy1) / (total2 - total1)."""
    idle, kernel, user = (wintypes.FILETIME() for _ in range(3))
    if not _kernel.GetSystemTimes(ctypes.byref(idle), ctypes.byref(kernel), ctypes.byref(user)):
        raise ctypes.WinError(ctypes.get_last_error())
    total = (_ticks(kernel) + _ticks(user)) / 1e7  # kernel time includes idle time
    return total - _ticks(idle) / 1e7, total


def service_pid(name):
    """Process id of a running Windows service (`sc queryex`), e.g. service_pid("Dnscache")."""
    output = subprocess.run(["sc", "queryex", name], capture_output=True, text=True, check=True).stdout
    match = re.search(r"PID\s*:\s*(\d+)", output)
    if not match or match.group(1) == "0":
        raise RuntimeError(f"service {name} is not running")
    return int(match.group(1))


if __name__ == "__main__":
    pid = service_pid("Dnscache")
    print(f"Dnscache pid {pid}, CPU so far {cpu_seconds([pid]).get(pid, 0):.2f} s")
