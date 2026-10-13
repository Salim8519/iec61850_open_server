"""Start the network helper with a Windows administrator (UAC) prompt and wait."""
import ctypes
import os
import subprocess
import sys
from ctypes import wintypes
from pathlib import Path

SEE_MASK_NOCLOSEPROCESS = 0x00000040
SW_HIDE = 0
ERROR_CANCELLED = 1223


class _ExecuteInfo(ctypes.Structure):
    _fields_ = [("cbSize", wintypes.DWORD), ("fMask", ctypes.c_ulong), ("hwnd", wintypes.HANDLE),
                ("lpVerb", wintypes.LPCWSTR), ("lpFile", wintypes.LPCWSTR),
                ("lpParameters", wintypes.LPCWSTR), ("lpDirectory", wintypes.LPCWSTR),
                ("nShow", ctypes.c_int), ("hInstApp", wintypes.HINSTANCE), ("lpIDList", ctypes.c_void_p),
                ("lpClass", wintypes.LPCWSTR), ("hkeyClass", wintypes.HKEY), ("dwHotKey", wintypes.DWORD),
                ("hIconOrMonitor", wintypes.HANDLE), ("hProcess", wintypes.HANDLE)]


class ElevationCancelled(Exception):
    """The user declined the Windows administrator prompt."""


def helper_command(request_path, digest, program):
    """Same helper code in both modes; only the way Python is started differs."""
    arguments = [str(request_path), digest, str(program)]
    if getattr(sys, "frozen", False):
        return sys.executable, ["--network-helper", *arguments]
    script = Path(__file__).resolve().parent / "helper" / "run.py"
    return sys.executable, [str(script), *arguments]


def run_elevated(executable, arguments, timeout_ms=120000):
    if os.name != "nt":
        raise RuntimeError("Lab network setup is only available on Windows")
    parameters = subprocess.list2cmdline(arguments)
    info = _ExecuteInfo(cbSize=ctypes.sizeof(_ExecuteInfo), fMask=SEE_MASK_NOCLOSEPROCESS,
                        lpVerb="runas", lpFile=str(executable), lpParameters=parameters, nShow=SW_HIDE)
    shell32 = ctypes.WinDLL("shell32", use_last_error=True)
    if not shell32.ShellExecuteExW(ctypes.byref(info)):
        error = ctypes.get_last_error()
        if error == ERROR_CANCELLED:
            raise ElevationCancelled("Windows administrator permission was not given, so nothing was changed")
        raise RuntimeError(f"Windows could not start the network helper (error {error})")
    kernel32 = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel32.WaitForSingleObject.argtypes = [wintypes.HANDLE, wintypes.DWORD]
    kernel32.GetExitCodeProcess.argtypes = [wintypes.HANDLE, ctypes.POINTER(wintypes.DWORD)]
    kernel32.CloseHandle.argtypes = [wintypes.HANDLE]
    try:
        if kernel32.WaitForSingleObject(info.hProcess, timeout_ms) != 0:
            raise RuntimeError("The network helper did not finish within two minutes")
        code = wintypes.DWORD()
        kernel32.GetExitCodeProcess(info.hProcess, ctypes.byref(code))
        return code.value
    finally:
        kernel32.CloseHandle(info.hProcess)
