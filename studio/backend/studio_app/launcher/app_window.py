"""Open the packaged app in a chromeless browser window (Edge, then Chrome, then default).

Packaged launches only: a double-clicked desktop app must show a window. Development
never opens windows (ADR-0012); its launcher prints the address instead.
"""
import logging
import os
from pathlib import Path
import subprocess
import tempfile
import threading
import webbrowser
from studio_app.launcher.window_focus import focus_process_window

_CANDIDATES = (
    ("ProgramFiles(x86)", "Microsoft/Edge/Application/msedge.exe"),
    ("ProgramFiles", "Microsoft/Edge/Application/msedge.exe"),
    ("ProgramFiles", "Google/Chrome/Application/chrome.exe"),
    ("ProgramFiles(x86)", "Google/Chrome/Application/chrome.exe"),
    ("LOCALAPPDATA", "Google/Chrome/Application/chrome.exe"),
)


def find_browser(environ=None):
    environ = os.environ if environ is None else environ
    for variable, relative in _CANDIDATES:
        base = environ.get(variable)
        if base and (Path(base) / relative).is_file():
            return Path(base) / relative
    return None


def app_window_command(browser, url, profile):
    return [str(browser), f"--app={url}", f"--user-data-dir={profile}", "--no-first-run",
            "--no-default-browser-check", "--window-size=1440,900"]


def profile_in_use(profile):
    """Chromium keeps <profile>/lockfile open while a browser runs on that profile."""
    try:
        (Path(profile) / "lockfile").unlink()
    except FileNotFoundError:
        return False
    except OSError:
        return True
    return False


def choose_profile(data_directory):
    """The window's own browser profile; a leftover window from an earlier run never adopts ours."""
    for name in ("browser-profile", "browser-profile-2", "browser-profile-3"):
        profile = Path(data_directory) / name
        try:
            profile.mkdir(parents=True, exist_ok=True)
        except OSError:
            break
        if not profile_in_use(profile):
            return profile
    return Path(tempfile.mkdtemp(prefix="ScadaBench-browser-"))


def open_app_window(url, profile):
    """Start the window; return its process, or None after falling back to the default browser."""
    browser = find_browser()
    if browser is None:
        webbrowser.open(url)
        return None
    flags = subprocess.CREATE_NEW_PROCESS_GROUP if os.name == "nt" else 0
    try:
        return subprocess.Popen(app_window_command(browser, url, profile), creationflags=flags,
                                stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, close_fds=True)
    except OSError:
        webbrowser.open(url)
        return None


def watch_window(process, on_closed, profile):
    """Stop ScadaBench when its own app window closes.

    If the browser exits while another browser still holds the profile, it handed the
    window to that browser; ScadaBench keeps running and a later launch reopens it.
    """
    def run():
        process.wait()
        if profile_in_use(profile):
            logging.getLogger(__name__).warning("App window was handed to another browser process")
            return
        on_closed()
    thread = threading.Thread(target=run, name="app-window", daemon=True)
    thread.start()
    return thread


class AppWindow:
    """The packaged app's single window: open it, show it again on request, stop when closed."""

    def __init__(self, url, data_directory, on_closed):
        self.url = url
        self.profile = choose_profile(data_directory)
        self.on_closed = on_closed
        self.process = None
        self.lock = threading.Lock()

    def open(self):
        self.process = open_app_window(self.url, self.profile)
        if self.process is not None:
            watch_window(self.process, self.on_closed, self.profile)
        return self

    def show(self):
        """Called when the user starts ScadaBench again: focus the window, or reopen it."""
        with self.lock:
            process = self.process
            if process is not None and process.poll() is None and focus_process_window(process.pid):
                return
            if process is not None and process.poll() is not None and not profile_in_use(self.profile):
                return  # The window was closed, so ScadaBench is already stopping.
            if process is None:
                webbrowser.open(self.url)
                return
            open_app_window(self.url, self.profile)  # Opens in the browser that holds the profile.
