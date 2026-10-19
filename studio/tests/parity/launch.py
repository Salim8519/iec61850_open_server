"""Spawn one engine process with the shared CLI contract and wait until it is ready.

<engine> --port 0 --no-browser --data-dir <data> --native-exe <exe> --frontend <dir>
         --parity-seams <tmp>/seams.json
Python (python-http) is `python -m parity.server`; Go is the parity build of the server.
Both print `ScadaBench: http://127.0.0.1:<port>`; the harness then requires the 401
"#session=" answer, so a half-started engine never reaches a test.
"""
import http.client
import json
import os
from pathlib import Path
import secrets
import shutil
import subprocess
import sys
import tempfile
import threading
import time

from parity import wire
from parity.jobobject import Job

STUDIO = Path(__file__).resolve().parents[2]
TESTS = STUDIO / "tests"
DEFAULT_GO_BIN = STUDIO / "engine/build/studio_server_parity.exe"
READY_PREFIX = "ScadaBench: http://"


def engine_name():
    return os.environ.get("STUDIO_ENGINE", "python")


def engine_command(engine):
    if engine == "python-http":
        return [sys.executable, "-m", "parity.server"]
    if engine == "go":
        binary = Path(os.environ.get("STUDIO_GO_BIN", DEFAULT_GO_BIN))
        if not binary.is_file():
            raise FileNotFoundError(f"Go parity binary not found: {binary} (set STUDIO_GO_BIN)")
        return [str(binary)]
    raise ValueError(f"STUDIO_ENGINE={engine!r} is not an HTTP engine (python-http or go)")


class EngineProcess:
    def __init__(self, engine, data_dir, native_exe, frontend, seams, port=0, env=None):
        self.engine, self.data_dir = engine, Path(data_dir)
        self.native_exe, self.frontend = Path(native_exe), Path(frontend)
        self.parity_key = secrets.token_hex(16)
        self.seams = {**seams, "parity_key": self.parity_key}
        self.requested_port, self.extra_env = port, dict(env or {})
        self.lines, self.url, self.port, self.process, self.job = [], None, None, None, None
        self.scratch = Path(tempfile.mkdtemp(prefix="studio-parity-"))
        self._ready = threading.Event()

    @property
    def pid(self):
        return self.process.pid if self.process else None

    def start(self, timeout=30):
        seams_file = self.scratch / "seams.json"
        seams_file.write_text(json.dumps(self.seams), encoding="utf-8")
        (self.scratch / "home").mkdir()
        args = ["--port", str(self.requested_port), "--no-browser", "--data-dir", str(self.data_dir),
                "--native-exe", str(self.native_exe), "--frontend", str(self.frontend),
                "--parity-seams", str(seams_file)]
        env = {**os.environ, "SCADABENCH_UPDATE_CHECK": "off", "LOCALAPPDATA": str(self.scratch / "home"),
               "PYTHONHASHSEED": os.environ.get("PYTHONHASHSEED", "0"),
               "PYTHONPATH": os.pathsep.join([str(STUDIO / "backend"), str(TESTS)]), **self.extra_env}
        self.job = Job()
        self.process = subprocess.Popen(engine_command(self.engine) + args, cwd=str(TESTS), env=env,
                                        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL,
                                        text=True, encoding="utf-8", errors="replace")
        self.job.assign(self.process)
        threading.Thread(target=self._read_output, name="parity-engine-output", daemon=True).start()
        if not self._ready.wait(timeout) or self.url is None:
            self.kill()
            raise RuntimeError(f"{self.engine} engine did not print its ready line within {timeout} s:\n"
                               + "".join(self.lines[-40:]))
        status, body = self.raw("GET", "/api/v1/project")
        if status != 401 or "#session=" not in json.loads(body).get("error", ""):
            self.kill()
            raise RuntimeError(f"{self.engine} engine answered {status} {body!r} instead of the 401 session error")
        return self

    def _read_output(self):
        for line in self.process.stdout:
            self.lines.append(line)
            if line.startswith(READY_PREFIX) and self.url is None:
                self.url = line.split(":", 1)[1].strip()
                self.port = int(self.url.rsplit(":", 1)[1])
                self._ready.set()
        self._ready.set()

    def raw(self, method, path, body=None, headers=None):
        connection = http.client.HTTPConnection("127.0.0.1", self.port, timeout=120)
        try:
            connection.request(method, path, body=body, headers=headers or {})
            response = connection.getresponse()
            return response.status, response.read()
        finally:
            connection.close()

    def parity(self, method, operation, body=None, **query):
        """Call /api/v1/_parity/<operation>; values use the tagged JSON of parity.wire."""
        path = "/api/v1/_parity/" + operation
        if query:
            path += "?" + "&".join(f"{key}={_quote(value)}" for key, value in query.items())
        data = None if body is None else wire.dumps(body).encode()
        status, raw = self.raw(method, path, data, {"X-Parity-Key": self.parity_key,
                                                    "Content-Type": "application/json"})
        answer = json.loads(raw) if raw else {}
        if status != 200:
            raise _engine_error(answer, status, operation)
        return wire.decode(answer.get("value"))

    def shutdown(self, wait=10):
        """Ask the engine to stop like the launcher does, then report surviving workers."""
        orphans = {}
        if self.process and self.process.poll() is None:
            try:
                self.parity("POST", "shutdown", {})
            except (OSError, RuntimeError, http.client.HTTPException):
                pass
            try:
                self.process.wait(wait)
            except subprocess.TimeoutExpired:
                orphans[self.process.pid] = f"{self.engine} engine (did not exit)"
        deadline, alive = time.monotonic() + 2, {}
        while self.job:
            alive = {pid: name for pid, name in self.job.processes().items() if name.lower().startswith("studio_")}
            if not alive or time.monotonic() >= deadline:
                break
            time.sleep(0.1)
        orphans.update(alive)
        self.kill()
        return orphans

    def kill(self):
        if self.job:
            self.job.close()
            self.job = None
        if self.process and self.process.poll() is None:
            self.process.kill()
        if self.process:
            try:
                self.process.wait(5)
            except subprocess.TimeoutExpired:
                pass
        shutil.rmtree(self.scratch, ignore_errors=True)


def _quote(value):
    from urllib.parse import quote
    return quote(str(value), safe="")


def _engine_error(answer, status, operation):
    if "raise" in answer:
        return wire.decode_exception(answer["raise"])
    return RuntimeError(f"parity {operation} answered {status}: {answer}")
