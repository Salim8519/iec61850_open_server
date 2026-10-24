"""Start a throwaway ScadaBench copy for recordings (the app built by video_app.py).

The Go engine (as released) serves the production frontend bundle with the real native
workers, on a free port and a temporary data folder. Not the packaged EXE, so there is
no license gate; product behaviour and defaults are the same.
"""
from __future__ import annotations

import shutil
import signal
import socket
import subprocess
import tempfile
import threading
from pathlib import Path

from video_app import APP


def free_port() -> int:
    with socket.socket() as probe:
        probe.bind(("127.0.0.1", 0))
        return probe.getsockname()[1]


def start() -> tuple[subprocess.Popen, str, Path]:
    """Start the app copy; return its process, browser address (with session) and data folder."""
    server = APP / "studio_server.exe"
    if not server.exists():
        raise RuntimeError("build the video app first: python marketing/videos/video_app.py")
    data = Path(tempfile.mkdtemp(prefix="scadabench-video-")) / "data"
    process = subprocess.Popen([str(server), "--no-browser", "--port", str(free_port()), "--data-dir", str(data),
                                "--native-exe", str(APP / "native" / "studio_ied.exe")],
                               stdout=subprocess.PIPE, text=True, creationflags=subprocess.CREATE_NEW_PROCESS_GROUP)
    for line in process.stdout:
        if line.startswith("Open in your browser: "):
            # keep reading so a chatty server never blocks on a full pipe
            threading.Thread(target=process.stdout.read, daemon=True).start()
            return process, line.split(": ", 1)[1].strip(), data
    process.wait()
    raise RuntimeError(f"the app copy did not start (exit code {process.returncode})")


def stop(process: subprocess.Popen, data: Path) -> None:
    """Stop like Ctrl+C, so the engine stops its IED workers, then remove the data folder."""
    try:
        process.send_signal(signal.CTRL_BREAK_EVENT)
        process.wait(timeout=20)
    except (OSError, subprocess.TimeoutExpired):
        process.kill()
        process.wait()
    shutil.rmtree(data.parent, ignore_errors=True)
