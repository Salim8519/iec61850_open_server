"""Isolated native worker rig for speed evidence (workstream G); never touches a running app.

Workers listen on 127.67.x.y with ports >= 20000 and use public/synthetic SCL only
(make_template, scd/open_substation.scd). Every process started here is stopped here.
"""
import ctypes
import json
import re
import socket
import subprocess
import sys
import threading
import time
from ctypes import wintypes
from pathlib import Path

STUDIO = Path(__file__).resolve().parents[1]
ROOT = STUDIO.parent
sys.path.insert(0, str(STUDIO / "backend"))
from studio_app.runtime.worker_process import WorkerProcess  # noqa: E402
from studio_app.scl.importer import import_scl  # noqa: E402
from studio_app.scl.templates import make_template  # noqa: E402

GOOSE_TIMING = b'<MinTime unit="s" multiplier="m">4</MinTime><MaxTime unit="s" multiplier="m">1000</MaxTime>'


def template_model(name, enhanced=False):
    source = make_template(name)
    if enhanced:  # enhanced-security SBO: select, operate and CommandTermination
        source = source.replace(b"<Val>3</Val>", b"<Val>4</Val>")
    return import_scl(source)[0]


def goose_model():
    """IED1_XCBR of the public sample with MinTime 4 ms / MaxTime 1000 ms (as in change 0102)."""
    source = (ROOT / "scd/open_substation.scd").read_bytes()
    pattern = rb'(<P type="APPID">1000</P>\s*</Address>)'
    source, found = re.subn(pattern, lambda match: match.group(1) + GOOSE_TIMING, source, count=1)
    assert found == 1, "sample SCD layout changed"
    return import_scl(source, "open_substation.scd", ["IED1_XCBR"])[0]


class Worker:
    def __init__(self, binary, model, folder, ip, port, goose_interface=None):
        self.events = []
        self.ip, self.port = ip, port
        config = Path(folder) / f"{ip}_{port}.cfg"
        config.write_text(model["cfg"], encoding="utf-8")
        self.process = WorkerProcess(Path(binary) / "studio_ied.exe", self.events.append)
        options = {"publish": True, "interface": goose_interface} if goose_interface else {}
        self.started = self.process.request("start", cfg=str(config), ip=ip, port=port,
                                            controls=model["controls"], edition=2, goose=options)

    def request(self, op, **values):
        return self.process.request(op, **values)

    def close(self):
        self.process.close()


class GooseSink:
    """Stands in for studio_goose.exe: receives the worker's GOOSE datagrams with arrival times."""

    def __init__(self):
        self.socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.socket.bind(("127.0.0.1", 0))
        self.socket.settimeout(0.2)
        self.port = self.socket.getsockname()[1]
        self.frames = []
        self.running = True
        self.thread = threading.Thread(target=self._receive, daemon=True)
        self.thread.start()

    def interface(self, mac="010ccd010001"):
        return f"udp:{self.port}:{mac}"

    def _receive(self):
        while self.running:
            try:
                data = self.socket.recv(65536)
            except socket.timeout:
                continue
            except OSError:
                return
            self.frames.append((time.perf_counter(), data))

    def close(self):
        self.running = False
        self.thread.join()
        self.socket.close()


def bench(binary, ip, port, mode, reference, fc, active, idle, count, timeout=600):
    arguments = [str(Path(binary) / "studio_test_mms_bench.exe"), ip, str(port), mode, reference, fc,
                 str(active), str(idle), str(count)]
    completed = subprocess.run(arguments, capture_output=True, text=True, timeout=timeout)
    if not completed.stdout.strip():
        raise RuntimeError(completed.stderr)
    return json.loads(completed.stdout.strip().splitlines()[-1])


def start_bench(binary, ip, port, reference, fc, active, idle, count):
    arguments = [str(Path(binary) / "studio_test_mms_bench.exe"), ip, str(port), "read", reference, fc,
                 str(active), str(idle), str(count)]
    return subprocess.Popen(arguments, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)


class FileTime(ctypes.Structure):
    _fields_ = [("low", wintypes.DWORD), ("high", wintypes.DWORD)]


def cpu_seconds(process):
    """Kernel + user CPU time of a Popen process (Windows)."""
    kernel = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel.GetProcessTimes.argtypes = [wintypes.HANDLE] + [ctypes.POINTER(FileTime)] * 4
    times = [FileTime() for _ in range(4)]
    if not kernel.GetProcessTimes(int(process._handle), *[ctypes.byref(item) for item in times]):
        raise ctypes.WinError(ctypes.get_last_error())
    return sum((item.high << 32 | item.low) for item in times[2:]) / 1e7
