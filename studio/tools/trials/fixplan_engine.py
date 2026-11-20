"""Shared harness for the fix-plan live checks (change 0226): one engine instance of its own.

Each check starts `studio/build/engine/studio_server.exe` with its own data folder, a port in
23000-23999 and the native workers of the main checkout (FIXPLAN_NATIVE overrides the folder).
The engine runs in its own hidden console (so Quit is a real Ctrl+C) and in a Job object, so
nothing it started survives the check. IED addresses are 127.67.23.x on loopback.

Not part of verify.py: run each fixplan_<item>.py by hand on Windows.
"""
import ctypes
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import threading
import time
import zipfile

STUDIO = Path(__file__).resolve().parents[2]
sys.path[:0] = [str(STUDIO / "tools"), str(STUDIO / "tests")]

from engine_bench.engine import Engine  # noqa: E402
from engine_bench.shutdown import hidden_console, send_ctrl_c  # noqa: E402

MAIN = Path(os.environ.get("FIXPLAN_MAIN", r"C:\Users\Salim\Desktop\Windsurf\iec61850_open_server"))
NATIVE = Path(os.environ.get("FIXPLAN_NATIVE", MAIN / "studio/native/build/Release"))
RANGE = "127.67.23"
RESULTS = []


class FixEngine(Engine):
    """An engine on a fixed port and a data folder that outlives one process (kill, start again)."""

    def __init__(self, port, data=None):
        super().__init__("go", NATIVE)
        self.port_wanted = port
        if data:
            self.data = Path(data)
        self.popen_options = hidden_console()

    def start(self, timeout=90):
        home = self.scratch / "home"
        home.mkdir(exist_ok=True)
        env = {**os.environ, "SCADABENCH_UPDATE_CHECK": "off", "LOCALAPPDATA": str(home)}
        args = ["--port", str(self.port_wanted), "--no-browser", "--data-dir", str(self.data),
                "--native-exe", str(self.native_bin / "studio_ied.exe")]
        from parity.jobobject import Job
        self.lines, self.token, self.cookie = [], None, ""
        self._ready = threading.Event()
        started = time.perf_counter()
        self.job = Job()
        self.process = subprocess.Popen([str(STUDIO / "build/engine/studio_server.exe")] + args, env=env,
                                        cwd=str(self.scratch), stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                        stdin=subprocess.DEVNULL, text=True, encoding="utf-8", errors="replace",
                                        **self.popen_options)
        self.job.assign(self.process)
        threading.Thread(target=self._read, daemon=True).start()
        if not self._ready.wait(timeout) or not self.token:
            raise RuntimeError("engine did not start:\n" + "".join(self.output(30)))
        self.ready_seconds = time.perf_counter() - started
        status, _, headers = self.request("POST", "/api/v1/session", {"token": self.token})
        if status != 200:
            raise RuntimeError(f"session refused: {status}")
        self.cookie = headers.get("Set-Cookie", "").split(";", 1)[0]
        return self

    def project(self):
        return self.json("GET", "/api/v1/project")

    def wait_project(self, timeout=60):
        """GET /project once start-up recovery finished (503 project_opening before that)."""
        deadline = time.time() + timeout
        while True:
            status, payload, _ = self.request("GET", "/api/v1/project")
            if status == 200:
                return json.loads(payload)
            if time.time() > deadline:
                raise RuntimeError(f"project did not open: {status} {payload[:300]!r}")
            time.sleep(0.2)

    def kill(self):
        """TerminateProcess on the engine only (taskkill /F); workers notice the closed pipe."""
        self.process.kill()
        self.process.wait(10)

    def quit(self, timeout=60):
        """Ctrl+C like closing the window; seconds until the engine process ended (None: still alive)."""
        started = time.perf_counter()
        if not send_ctrl_c(self.process.pid):
            raise RuntimeError("Ctrl+C could not be sent")
        try:
            self.process.wait(timeout)
        except subprocess.TimeoutExpired:
            return None
        return round(time.perf_counter() - started, 3)

    def close(self, keep_data=True):
        """Kill whatever is left (Job object) and keep the data folder for the next start."""
        if self.job:
            self.job.close()
            self.job = None
        if self.process and self.process.poll() is None:
            self.process.kill()
            self.process.wait(10)

    def workers(self, name=None):
        """{pid: image} of the native programs this engine started (optionally one image name)."""
        found = self.children() if self.job else {}
        return {pid: image for pid, image in found.items() if name is None or image.lower() == name.lower()}


class Events:
    """The engine's event stream (GET /api/v1/events) collected in a background thread."""

    def __init__(self, engine):
        import http.client
        self.items, self.connection = [], http.client.HTTPConnection("127.0.0.1", engine.port, timeout=3600)
        self.connection.request("GET", "/api/v1/events", headers={"Host": f"127.0.0.1:{engine.port}",
                                                                 "Cookie": engine.cookie})
        self.response = self.connection.getresponse()
        threading.Thread(target=self._read, daemon=True).start()

    def _read(self):
        try:
            for raw in self.response:
                line = raw.decode("utf-8", "replace").strip()
                if line.startswith("data:"):
                    try:
                        item = json.loads(line[5:])
                    except ValueError:
                        continue
                    item["_at"] = time.perf_counter()
                    self.items.append(item)
        except (OSError, ValueError):
            pass

    def of(self, kind, since=0.0):
        return [item for item in list(self.items) if item.get("kind") == kind and item["_at"] >= since]

    def close(self):
        try:
            self.connection.sock.close()
        except (OSError, AttributeError):
            pass


def thread_count(pid):
    """Threads of one process (Toolhelp snapshot), to see connection threads that never end."""
    from ctypes import wintypes

    class Entry(ctypes.Structure):
        _fields_ = [("dwSize", wintypes.DWORD), ("cntUsage", wintypes.DWORD), ("th32ThreadID", wintypes.DWORD),
                    ("th32OwnerProcessID", wintypes.DWORD), ("tpBasePri", ctypes.c_long),
                    ("tpDeltaPri", ctypes.c_long), ("dwFlags", wintypes.DWORD)]
    kernel = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel.CreateToolhelp32Snapshot.restype = ctypes.c_void_p
    snapshot = kernel.CreateToolhelp32Snapshot(0x4, 0)  # TH32CS_SNAPTHREAD
    entry, count = Entry(), 0
    entry.dwSize = ctypes.sizeof(Entry)
    try:
        more = kernel.Thread32First(ctypes.c_void_p(snapshot), ctypes.byref(entry))
        while more:
            count += entry.th32OwnerProcessID == pid
            more = kernel.Thread32Next(ctypes.c_void_p(snapshot), ctypes.byref(entry))
    finally:
        kernel.CloseHandle(ctypes.c_void_p(snapshot))
    return count


def alive(pid):
    kernel = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel.OpenProcess.restype = ctypes.c_void_p
    handle = kernel.OpenProcess(0x101000, False, pid)  # QUERY_LIMITED_INFORMATION | SYNCHRONIZE
    if not handle:
        return False
    try:
        return kernel.WaitForSingleObject(ctypes.c_void_p(handle), 0) == 0x102  # WAIT_TIMEOUT
    finally:
        kernel.CloseHandle(ctypes.c_void_p(handle))


def ntdll_call(function, pid):
    kernel, ntdll = ctypes.WinDLL("kernel32", use_last_error=True), ctypes.WinDLL("ntdll")
    kernel.OpenProcess.restype = ctypes.c_void_p
    handle = kernel.OpenProcess(0x0800, False, pid)  # PROCESS_SUSPEND_RESUME
    if not handle:
        raise ctypes.WinError(ctypes.get_last_error())
    try:
        status = getattr(ntdll, function)(ctypes.c_void_p(handle))
        if status != 0:
            raise OSError(f"{function}({pid}) returned 0x{status & 0xffffffff:08x}")
    finally:
        kernel.CloseHandle(ctypes.c_void_p(handle))


def suspend(pid):
    """Freeze a process like Task Manager's 'Suspend' (NtSuspendProcess)."""
    ntdll_call("NtSuspendProcess", pid)


def resume(pid):
    ntdll_call("NtResumeProcess", pid)


def scratch_dir(name):
    return Path(tempfile.mkdtemp(prefix=f"fixplan-{name}-"))


def add_template(engine, name):
    return engine.json("POST", "/api/v1/ieds/template", {"name": name})


def import_scl(engine, filename, data, names):
    engine.upload("/api/v1/scl/preview", "file", filename, data)
    return engine.upload("/api/v1/scl/import", "file", filename, data, {"selected_names": json.dumps(names)})


def rewrite_project(engine, change):
    """Download the project, let change(document) edit project.json, open it again (stopped only)."""
    status, archive, _ = engine.request("GET", "/api/v1/project/download")
    if status != 200:
        raise RuntimeError(f"download failed: {status}")
    with zipfile.ZipFile(io.BytesIO(archive)) as source:
        files = {name: source.read(name) for name in source.namelist()}
    document = json.loads(files["project.json"])
    change(document)
    files["project.json"] = json.dumps(document).encode()
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as target:
        for name, data in files.items():
            target.writestr(name, data)
    return engine.upload("/api/v1/project/open", "file", "fixplan.scadabench", stream.getvalue())


def place(document, addresses):
    """Give each IED (by name) an address and port: {name: (ip, port)}."""
    for ied in document["ieds"]:
        if ied["name"] in addresses:
            ied["ip"], ied["port"] = addresses[ied["name"]]


def wait_for(predicate, timeout, pause=0.1):
    deadline = time.perf_counter() + timeout
    while time.perf_counter() < deadline:
        value = predicate()
        if value:
            return value
        time.sleep(pause)
    return predicate()


def check(item, name, ok, evidence, owner=False):
    """Record one result line (printed now, summarised by finish()). owner=True: a behaviour the
    owner must decide on (fix plan "not decided"); it is reported as OWNER, never as a failure."""
    verdict = "OWNER" if owner else ("PASS" if ok else "FAIL")
    RESULTS.append({"item": item, "check": name, "verdict": verdict, "evidence": evidence})
    print(f"[{verdict}] {item} {name}: {json.dumps(evidence, default=str)}", flush=True)
    return ok


def finish(label):
    """Print the summary and write it next to the main checkout's other trial output."""
    out = MAIN / ".local" / "fixplan"
    out.mkdir(parents=True, exist_ok=True)
    stamp = time.strftime("%Y%m%d-%H%M%S")
    (out / f"{label}-{stamp}.json").write_text(json.dumps(RESULTS, indent=1, default=str), encoding="utf-8")
    failed = [entry for entry in RESULTS if entry["verdict"] == "FAIL"]
    owner = [entry for entry in RESULTS if entry["verdict"] == "OWNER"]
    print(f"{label}: {len(RESULTS) - len(failed) - len(owner)} passed, {len(failed)} failed, "
          f"{len(owner)} for the owner to decide", flush=True)
    return 1 if failed else 0
