"""One ScadaBench per data folder: a second launch asks the running one to show its window.

The data-folder lease decides who runs. This module only carries the "show yourself"
request between the two processes through a per-session named Windows event, so no
token, port or file is shared and the request cannot change anything else.
"""
import hashlib
import os
import threading

EVENT_MODIFY_STATE = 0x0002
INFINITE = 0xFFFFFFFF
ASFW_ANY = -1


def event_name(directory):
    folder = os.path.normcase(os.path.abspath(str(directory)))
    return "Local\\ScadaBench-Show-" + hashlib.sha256(folder.encode("utf-8")).hexdigest()[:24]


def _kernel32():
    import ctypes
    from ctypes import wintypes
    kernel32 = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel32.CreateEventW.restype = wintypes.HANDLE
    kernel32.CreateEventW.argtypes = (wintypes.LPVOID, wintypes.BOOL, wintypes.BOOL, wintypes.LPCWSTR)
    kernel32.OpenEventW.restype = wintypes.HANDLE
    kernel32.OpenEventW.argtypes = (wintypes.DWORD, wintypes.BOOL, wintypes.LPCWSTR)
    kernel32.SetEvent.argtypes = (wintypes.HANDLE,)
    kernel32.WaitForSingleObject.argtypes = (wintypes.HANDLE, wintypes.DWORD)
    kernel32.WaitForSingleObject.restype = wintypes.DWORD
    kernel32.CloseHandle.argtypes = (wintypes.HANDLE,)
    return kernel32


def request_show(directory):
    """Ask the running instance to bring its window forward. True when it was asked."""
    if os.name != "nt":
        return False
    import ctypes
    kernel32 = _kernel32()
    handle = kernel32.OpenEventW(EVENT_MODIFY_STATE, False, event_name(directory))
    if not handle:
        return False
    try:
        # This process was just started by the user, so it may hand its right to take
        # the foreground to the running instance; Windows blocks focus stealing otherwise.
        ctypes.windll.user32.AllowSetForegroundWindow(ASFW_ANY)
        return bool(kernel32.SetEvent(handle))
    finally:
        kernel32.CloseHandle(handle)


class ShowListener:
    """Calls on_show each time another launch asks this instance to show itself."""

    def __init__(self, directory, on_show):
        self.directory = directory
        self.on_show = on_show
        self.handle = None
        self.thread = None

    def start(self):
        if os.name != "nt":
            return self
        kernel32 = _kernel32()
        self.handle = kernel32.CreateEventW(None, False, False, event_name(self.directory))
        if not self.handle:
            return self
        self.thread = threading.Thread(target=self._run, args=(kernel32, self.handle),
                                       name="show-requests", daemon=True)
        self.thread.start()
        return self

    def _run(self, kernel32, handle):
        while kernel32.WaitForSingleObject(handle, INFINITE) == 0:
            if self.handle is None:
                return
            try:
                self.on_show()
            except Exception:  # A failed focus must never stop the running app.
                import logging
                logging.getLogger(__name__).exception("Could not show the ScadaBench window")

    def close(self):
        handle, self.handle = self.handle, None
        if handle:
            kernel32 = _kernel32()
            kernel32.SetEvent(handle)
            kernel32.CloseHandle(handle)
