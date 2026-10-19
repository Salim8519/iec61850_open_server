"""Per-process working set, private bytes, CPU time and handle count (Win32, read-only)."""
import ctypes
from ctypes import wintypes
import functools
import threading
import time

QUERY_LIMITED_INFORMATION = 0x1000
VM_READ = 0x0010


class _Counters(ctypes.Structure):
    _fields_ = [("cb", wintypes.DWORD), ("PageFaultCount", wintypes.DWORD),
                ("PeakWorkingSetSize", ctypes.c_size_t), ("WorkingSetSize", ctypes.c_size_t),
                ("QuotaPeakPagedPoolUsage", ctypes.c_size_t), ("QuotaPagedPoolUsage", ctypes.c_size_t),
                ("QuotaPeakNonPagedPoolUsage", ctypes.c_size_t), ("QuotaNonPagedPoolUsage", ctypes.c_size_t),
                ("PagefileUsage", ctypes.c_size_t), ("PeakPagefileUsage", ctypes.c_size_t),
                ("PrivateUsage", ctypes.c_size_t)]


@functools.lru_cache(maxsize=1)
def _api():
    kernel = ctypes.WinDLL("kernel32", use_last_error=True)
    psapi = ctypes.WinDLL("psapi", use_last_error=True)
    kernel.OpenProcess.restype = wintypes.HANDLE
    kernel.OpenProcess.argtypes = [wintypes.DWORD, wintypes.BOOL, wintypes.DWORD]
    kernel.GetProcessTimes.argtypes = [wintypes.HANDLE] + [ctypes.POINTER(wintypes.FILETIME)] * 4
    kernel.GetProcessHandleCount.argtypes = [wintypes.HANDLE, ctypes.POINTER(wintypes.DWORD)]
    kernel.CloseHandle.argtypes = [wintypes.HANDLE]
    psapi.GetProcessMemoryInfo.argtypes = [wintypes.HANDLE, ctypes.POINTER(_Counters), wintypes.DWORD]
    return kernel, psapi


def _seconds(filetime):
    return ((filetime.dwHighDateTime << 32) | filetime.dwLowDateTime) / 1e7


def sample(pid):
    """{ws, private, peak_ws, cpu (s), handles} or None when the process is gone."""
    _kernel, _psapi = _api()
    handle = _kernel.OpenProcess(QUERY_LIMITED_INFORMATION | VM_READ, False, pid)
    if not handle:
        return None
    try:
        counters = _Counters()
        counters.cb = ctypes.sizeof(counters)
        times = [wintypes.FILETIME() for _ in range(4)]
        handles = wintypes.DWORD()
        if not _psapi.GetProcessMemoryInfo(handle, ctypes.byref(counters), counters.cb):
            return None
        _kernel.GetProcessTimes(handle, *[ctypes.byref(item) for item in times])
        _kernel.GetProcessHandleCount(handle, ctypes.byref(handles))
        return {"ws": counters.WorkingSetSize, "private": counters.PrivateUsage,
                "peak_ws": counters.PeakWorkingSetSize, "cpu": _seconds(times[2]) + _seconds(times[3]),
                "handles": handles.value}
    finally:
        _kernel.CloseHandle(handle)


class Sampler:
    """Samples every process of a job once per `interval`. The backend group is every process
    with one of `backend_names` (a venv python.exe is a launcher stub plus the interpreter)."""

    def __init__(self, job, backend_names, interval=1.0):
        self.job, self.backend_names, self.interval = job, {name.lower() for name in backend_names}, interval
        self.rows, self._stop = [], threading.Event()
        self._thread = threading.Thread(target=self._run, name="bench-sampler", daemon=True)

    def start(self):
        self._thread.start()
        return self

    def _run(self):
        while not self._stop.is_set():
            at = time.monotonic()
            for pid, name in self.job.processes().items():
                stats = sample(pid)
                if stats:
                    self.rows.append({"t": at, "pid": pid, "name": name, "backend": name.lower() in self.backend_names, **stats})
            self._stop.wait(self.interval)

    def stop(self):
        self._stop.set()
        self._thread.join(5)
        return summarise(self.rows)


def summarise(rows):
    """Backend and children: peak/mean working set and private bytes, mean CPU % of one core."""
    result = {}
    for group, chosen in (("backend", [row for row in rows if row["backend"]]),
                          ("children", [row for row in rows if not row["backend"]])):
        times = sorted({row["t"] for row in chosen})
        totals = [sum(row["ws"] for row in chosen if row["t"] == t) for t in times]
        private = [sum(row["private"] for row in chosen if row["t"] == t) for t in times]
        cpu = {}
        for row in chosen:
            first, last = cpu.get(row["pid"], (row, row))
            cpu[row["pid"]] = (first if first["t"] <= row["t"] else row, row if row["t"] >= last["t"] else last)
        span = (times[-1] - times[0]) if len(times) > 1 else 0
        used = sum(last["cpu"] - first["cpu"] for first, last in cpu.values())
        result[group] = {"samples": len(times), "processes": len(cpu),
                         "ws_peak_mb": round(max(totals, default=0) / 2**20, 1),
                         "ws_mean_mb": round(sum(totals) / len(totals) / 2**20, 1) if totals else 0,
                         "private_peak_mb": round(max(private, default=0) / 2**20, 1),
                         "cpu_percent_one_core": round(100 * used / span, 2) if span else None,
                         "handles_last": sum(row["handles"] for row in chosen if times and row["t"] == times[-1])}
    return result
