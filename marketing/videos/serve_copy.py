"""Serve a throwaway ScadaBench copy for recordings (run with studio/.venv Python).

Prints the browser address and the data folder, then serves until killed. Development
build: no license gate, temporary project data, the real native IED workers.
"""
import secrets
import socket
import sys
import tempfile
from pathlib import Path

STUDIO = Path(__file__).resolve().parents[2] / "studio"
sys.path.insert(0, str(STUDIO / "backend"))

from waitress import serve  # noqa: E402
from studio_app.app_factory import create_app  # noqa: E402
from studio_app.settings import Settings  # noqa: E402

data = Path(tempfile.mkdtemp(prefix="scadabench-video-")) / "data"
session = secrets.token_hex(8)
with socket.socket() as probe:
    probe.bind(("127.0.0.1", 0))
    port = probe.getsockname()[1]
app = create_app(Settings(data, STUDIO / "native/build/Release/studio_ied.exe", STUDIO / "frontend/dist"), session)
print(f"http://127.0.0.1:{port}/#session={session}", flush=True)
print(data, flush=True)
serve(app, host="127.0.0.1", port=port, threads=8, _quiet=True)
