"""Bring the app window owned by one browser process to the front (restoring it if minimised)."""
import os

SW_RESTORE = 9


def _visible_windows(pid):
    import ctypes
    from ctypes import wintypes
    user32 = ctypes.windll.user32
    found = []

    @ctypes.WINFUNCTYPE(wintypes.BOOL, wintypes.HWND, wintypes.LPARAM)
    def collect(window, _param):
        owner = wintypes.DWORD()
        user32.GetWindowThreadProcessId(window, ctypes.byref(owner))
        if owner.value == pid and user32.IsWindowVisible(window) and not user32.GetWindow(window, 4):
            found.append(window)  # GW_OWNER (4): skip pop-ups owned by the main window.
        return True

    user32.EnumWindows(collect, 0)
    return found


def focus_process_window(pid):
    """True when a visible top-level window of process pid was brought forward."""
    if os.name != "nt" or not pid:
        return False
    import ctypes
    user32 = ctypes.windll.user32
    windows = _visible_windows(pid)
    if not windows:
        return False
    window = windows[0]
    if user32.IsIconic(window):
        user32.ShowWindow(window, SW_RESTORE)
    user32.SetForegroundWindow(window)
    return True
