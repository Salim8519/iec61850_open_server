"""One clock for every trial observation: Windows' precise wall time in Unix seconds.

The engine's event timestamps (Go time.Now), Windows' address creation times and process
creation/exit times (FILETIME) are all wall times, so the harness compares them on this one
scale. Python 3.12's time.time() on Windows ticks every ~15.6 ms; this clock does not.
"""
import ctypes
from ctypes import wintypes

EPOCH_DIFFERENCE = 11644473600  # seconds from 1601-01-01 (FILETIME) to 1970-01-01 (Unix)

_kernel = ctypes.WinDLL("kernel32")
_kernel.GetSystemTimePreciseAsFileTime.argtypes = [ctypes.POINTER(wintypes.FILETIME)]


def filetime_to_unix(value):
    """FILETIME ticks (100 ns since 1601) to Unix seconds; 0 stays 0 (no time)."""
    return value / 1e7 - EPOCH_DIFFERENCE if value else 0.0


def now():
    """Precise wall time in Unix seconds."""
    stamp = wintypes.FILETIME()
    _kernel.GetSystemTimePreciseAsFileTime(ctypes.byref(stamp))
    return filetime_to_unix((stamp.dwHighDateTime << 32) | stamp.dwLowDateTime)
