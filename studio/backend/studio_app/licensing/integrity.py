"""Tamper checks for packaged builds: shipped native workers and attached debuggers.

EXPECTED is empty in source (development never checks). compile_protected.py writes
the SHA-256 of each shipped native executable into the compiled copy, so a patched or
swapped worker makes the packaged app refuse to run until it is reinstalled.
"""
import hashlib
import os
from pathlib import Path

EXPECTED = {}

TAMPERED = "ScadaBench program files were changed or damaged. Reinstall ScadaBench."
DEBUGGER = "ScadaBench cannot run while a debugger is attached."


def file_digest(path):
    digest = hashlib.sha256()
    with open(path, "rb") as stream:
        for block in iter(lambda: stream.read(1 << 20), b""):
            digest.update(block)
    return digest.hexdigest()


def debugger_attached():
    if os.name != "nt":
        return False
    import ctypes
    kernel32 = ctypes.windll.kernel32
    remote = ctypes.c_int(0)
    kernel32.CheckRemoteDebuggerPresent(kernel32.GetCurrentProcess(), ctypes.byref(remote))
    return bool(kernel32.IsDebuggerPresent() or remote.value)


def problem(native_folder, expected=None):
    """Return a user-readable reason, or None when the installation is intact."""
    expected = EXPECTED if expected is None else expected
    if debugger_attached():
        return DEBUGGER
    for name, wanted in expected.items():
        try:
            if file_digest(Path(native_folder) / name) != wanted:
                return TAMPERED
        except OSError:
            return TAMPERED
    return None
