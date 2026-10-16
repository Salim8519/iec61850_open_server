"""Output for the windowed EXE, which has no console window.

Users see plain message boxes; diagnostics go to a dated log file; a launch from a
terminal (for example `ScadaBench.exe --self-test`) still prints to that terminal.
"""
import datetime
import logging
import os
import sys

ATTACH_PARENT_PROCESS = -1
MB_ICONINFORMATION = 0x40
MB_ICONERROR = 0x10


def headless():
    """True when nobody can read stdout/stderr (double-clicked windowed EXE)."""
    return sys.stdout is None or sys.stderr is None


def attach_parent_terminal():
    """Reuse the terminal that started this windowed process, if there is one."""
    if os.name != "nt" or not headless():
        return
    import ctypes
    if not ctypes.windll.kernel32.AttachConsole(ATTACH_PARENT_PROCESS):
        return
    try:
        stream = open("CONOUT$", "w", encoding="utf-8", errors="replace")
    except OSError:
        return
    sys.stdout = sys.stdout or stream
    sys.stderr = sys.stderr or stream


def configure_logging(directory):
    """Log to the terminal when there is one, otherwise append to data/logs/launcher-<date>.log."""
    if not headless():
        logging.basicConfig(level=logging.INFO)
        return
    folder = directory / "logs"
    try:
        folder.mkdir(parents=True, exist_ok=True)
        name = folder / f"launcher-{datetime.date.today():%Y%m%d}.log"
        logging.basicConfig(level=logging.INFO, filename=name, encoding="utf-8",
                            format="%(asctime)s %(levelname)s %(name)s: %(message)s")
    except OSError:
        logging.basicConfig(level=logging.INFO, handlers=[logging.NullHandler()])


def tell_user(text, error=False):
    """Show text to the user: in the terminal when there is one, otherwise in a message box."""
    if not headless():
        print(text, file=sys.stderr if error else sys.stdout, flush=True)
        return
    if os.name == "nt":
        import ctypes
        icon = MB_ICONERROR if error else MB_ICONINFORMATION
        ctypes.windll.user32.MessageBoxW(None, text, "ScadaBench", icon)
