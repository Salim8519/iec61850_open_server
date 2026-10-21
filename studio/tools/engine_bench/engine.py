"""Start the product launcher of either engine for a benchmark and talk to it over HTTP.

Python: `python -m studio_app.launcher.entrypoint --port 0 --no-browser --data-dir <tmp>
--native-exe <bin>/studio_ied.exe` (no --dev, so no file watcher). Go: the product server
binary with the same arguments. The launcher prints `Open in your browser: <url>#session=<token>`;
the benchmark signs in with that token like the browser does. Both run inside a Job object so
every worker is sampled and nothing survives the run.
"""
import http.client
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import threading
import time

from parity.jobobject import Job

STUDIO = Path(__file__).resolve().parents[2]
DEFAULT_GO = STUDIO / "build/engine/studio_server.exe"  # where run_dev_engine.py --engine go builds it


def command(engine, python):
    if engine == "python":
        return [python, "-m", "studio_app.launcher.entrypoint"]
    if engine == "go":
        binary = Path(os.environ.get("STUDIO_GO_SERVER", DEFAULT_GO))
        if not binary.is_file():
            raise FileNotFoundError(f"Go server not found: {binary} (set STUDIO_GO_SERVER)")
        return [str(binary)]
    raise ValueError(f"unknown engine {engine!r}")


class Engine:
    def __init__(self, engine, native_bin, python=sys.executable):
        self.engine, self.native_bin, self.python = engine, Path(native_bin), python
        self.scratch = Path(tempfile.mkdtemp(prefix="studio-bench-"))
        self.data = self.scratch / "data"
        self.url = self.token = self.port = self.process = self.job = None
        self.cookie, self.lines, self._ready = "", [], threading.Event()
        self.popen_options = {}  # lifecycle runs give the launcher its own hidden console (Ctrl+C)

    def start(self, timeout=60):
        (self.scratch / "home").mkdir()
        env = {**os.environ, "SCADABENCH_UPDATE_CHECK": "off", "LOCALAPPDATA": str(self.scratch / "home"),
               "PYTHONPATH": str(STUDIO / "backend")}
        args = ["--port", "0", "--no-browser", "--data-dir", str(self.data),
                "--native-exe", str(self.native_bin / "studio_ied.exe")]
        started = time.perf_counter()
        self.job = Job()
        self.process = subprocess.Popen(command(self.engine, self.python) + args, env=env, cwd=str(self.scratch),
                                        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL,
                                        text=True, encoding="utf-8", errors="replace", **self.popen_options)
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

    def output(self, count):
        """The launcher's last `count` lines with the session token hidden (safe for logs)."""
        return [line.split("#session=", 1)[0] + "#session=<hidden>\n" if "#session=" in line else line
                for line in self.lines[-count:]]

    def _read(self):
        for line in self.process.stdout:
            self.lines.append(line)
            if line.startswith("Open in your browser: ") and "#session=" in line:
                self.url, self.token = line.split(": ", 1)[1].strip().split("#session=")
                self.port = int(self.url.rsplit(":", 1)[1])
                self._ready.set()
        self._ready.set()

    def request(self, method, path, body=None, raw=None, content_type=None, timeout=600):
        headers = {"Host": f"127.0.0.1:{self.port}"}
        if self.cookie:
            headers["Cookie"] = self.cookie
        data = raw
        if body is not None:
            data, content_type = json.dumps(body).encode(), "application/json"
        if content_type:
            headers["Content-Type"] = content_type
        connection = http.client.HTTPConnection("127.0.0.1", self.port, timeout=timeout)
        try:
            connection.request(method, path, body=data, headers=headers)
            response = connection.getresponse()
            payload = response.read()
            return response.status, payload, dict(response.getheaders())
        finally:
            connection.close()

    def json(self, method, path, body=None, expect=200, **options):
        status, payload, _ = self.request(method, path, body, **options)
        if status != expect:
            raise RuntimeError(f"{method} {path} answered {status}: {payload[:300]!r}")
        return json.loads(payload) if payload else None

    def upload(self, path, field, filename, data, form=None, expect=200):
        boundary = "----bench" + os.urandom(8).hex()
        parts = [f'--{boundary}\r\nContent-Disposition: form-data; name="{name}"\r\n\r\n{value}\r\n'.encode()
                 for name, value in (form or {}).items()]
        parts.append(f'--{boundary}\r\nContent-Disposition: form-data; name="{field}"; filename="{filename}"\r\n'
                     f"Content-Type: application/octet-stream\r\n\r\n".encode() + data + b"\r\n")
        raw = b"".join(parts) + f"--{boundary}--\r\n".encode()
        return self.json("POST", path, raw=raw, content_type=f"multipart/form-data; boundary={boundary}", expect=expect)

    @property
    def backend_names(self):
        return {"python.exe", "pythonw.exe"} if self.engine == "python" else {Path(command("go", "")[0]).name}

    def children(self):
        return {pid: name for pid, name in self.job.processes().items() if name.lower() not in self.backend_names}

    def stop(self):
        """Stop the run over HTTP, then close the job (the launcher has no exit endpoint)."""
        if self.process and self.process.poll() is None and self.token:
            try:
                self.request("POST", "/api/v1/runtime/stop", timeout=60)
            except OSError:
                pass
        if self.job:
            self.job.close()
            self.job = None
        if self.process:
            try:
                self.process.wait(10)
            except subprocess.TimeoutExpired:
                self.process.kill()
        shutil.rmtree(self.scratch, ignore_errors=True)
