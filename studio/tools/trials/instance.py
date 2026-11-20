"""A separate engine instance for trials: own data folder, own port >= 20000, signed in.

It never touches the owner's installed app or its data folder. The Go server runs without
--dev (packaged behaviour) with the development native workers, inside a Job object so every
worker dies with it. The engine prints `Open in your browser: <url>#session=<token>`; the
instance signs in with that token like the browser does and keeps the session cookie.

    from instance import Instance
    with Instance() as engine:          # or Instance(port=20500, keep=True)
    # It refuses to start while another engine runs or the Lab card holds addresses
    # (busy_reasons); Instance(shared=True) overrides that for trials that need it.
        engine.json("GET", "/api/v1/project")
"""
import os
from pathlib import Path
import shutil
import socket
import subprocess
import sys
import threading
import time
from uuid import uuid4

TOOLS = Path(__file__).resolve().parents[1]
STUDIO = TOOLS.parent
for extra in (TOOLS, STUDIO / "tests"):
    if str(extra) not in sys.path:
        sys.path.insert(0, str(extra))

from engine_bench.engine import Engine  # noqa: E402  (HTTP helpers, token reading, stop)
from parity.jobobject import Job  # noqa: E402
import winaddr  # noqa: E402  (this folder: the scripts here run with it as sys.path[0])

SERVER = STUDIO / "build/engine/studio_server.exe"
NATIVE = Path(os.environ.get("SCADABENCH_TRIAL_NATIVE", STUDIO / "native/build/Release"))
ROOT = STUDIO.parent / ".local/trials/instances"  # inside the project: the antivirus allows it


def busy_reasons():
    """Why a trial must not start now. Another ScadaBench engine (the owner's app or another
    agent's instance) shares the one Lab card: every Stop of either one takes off the card the
    addresses the other one put there (Release keeps only its own), so both runs break."""
    reasons = []
    listing = subprocess.run(["tasklist", "/fi", "imagename eq studio_server.exe", "/fo", "csv", "/nh"],
                             capture_output=True, text=True).stdout
    others = [line.split(",")[1].strip('"') for line in listing.splitlines() if line.startswith('"studio_server')]
    if others:
        reasons.append(f"another studio_server.exe runs (pid {', '.join(others)})")
    card = winaddr.lab_card_index()
    foreign = [ip for ip in winaddr.addresses(interface=card) if not ip.startswith("169.254.")] if card else []
    if foreign:
        reasons.append(f"the Lab card already holds {len(foreign)} address(es), e.g. {foreign[0]}")
    return reasons


PORTS = [int(p) for p in os.environ.get("SCADABENCH_TRIAL_PORTS", "20000-20999").split("-")]


def free_port(start=PORTS[0], stop=PORTS[1]):
    """First port in [start, stop] that nothing listens on (both 127.0.0.1 binds succeed)."""
    for port in range(start, stop + 1):
        with socket.socket() as probe:
            try:
                probe.bind(("127.0.0.1", port))
                return port
            except OSError:
                continue
    raise RuntimeError(f"no free port in {start}-{stop}")


class Instance(Engine):
    def __init__(self, port=None, native=NATIVE, server=SERVER, keep=False, shared=False):
        super().__init__("go", native)
        self.shared = shared  # True: start even when busy_reasons() finds another engine
        shutil.rmtree(self.scratch, ignore_errors=True)  # the base class made one in %TEMP%
        self.scratch = ROOT / uuid4().hex[:12]
        self.data = self.scratch / "data"
        self.server, self.keep, self.wanted_port = Path(server), keep, port

    def start(self, timeout=60):
        reasons = [] if self.shared else busy_reasons()
        if reasons:
            raise RuntimeError("not starting a trial instance: " + "; ".join(reasons))
        if not self.server.is_file():
            raise FileNotFoundError(f"{self.server} missing: run go build in studio/engine first")
        (self.scratch / "home").mkdir(parents=True)
        port = self.wanted_port or free_port()
        if port < 20000:
            raise ValueError("trial instances use ports >= 20000")
        env = {**os.environ, "SCADABENCH_UPDATE_CHECK": "off", "LOCALAPPDATA": str(self.scratch / "home")}
        args = [str(self.server), "--port", str(port), "--no-browser", "--data-dir", str(self.data),
                "--native-exe", str(self.native_bin / "studio_ied.exe")]
        started = time.perf_counter()
        self.job = Job()
        self.process = subprocess.Popen(args, env=env, cwd=str(self.scratch), stdout=subprocess.PIPE,
                                        stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL, text=True,
                                        encoding="utf-8", errors="replace")
        self.job.assign(self.process)
        threading.Thread(target=self._read, daemon=True).start()
        if not self._ready.wait(timeout) or not self.token:
            self.stop()
            raise RuntimeError("engine did not start:\n" + "".join(self.output(30)))
        self.ready_seconds = time.perf_counter() - started
        status, _, headers = self.request("POST", "/api/v1/session", {"token": self.token})
        if status != 200:
            raise RuntimeError(f"session refused: {status}")
        self.cookie = headers.get("Set-Cookie", "").split(";", 1)[0]
        return self

    def stop(self):
        """Stop the run, close the job (kills every worker) and remove the folder unless kept."""
        scratch, keep = self.scratch, self.keep
        if keep:
            self.scratch = Path(os.devnull)  # the base stop removes self.scratch
        super().stop()
        self.scratch = scratch
        if not keep:
            shutil.rmtree(scratch, ignore_errors=True)

    def __enter__(self):
        return self.start()

    def __exit__(self, *_):
        self.stop()
