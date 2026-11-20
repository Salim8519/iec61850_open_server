"""Trial R3: what the spare IED processes cost, and what a packaged licence check costs per process.

    python studio/tools/trials/r3_spare_cost.py --server <trial exe> --native <trial native dir>

1. Starts the trial engine (temp data folder), opens the large project and waits until one
   spare studio_ied per simulated IED waits; then reads their private memory and their
   processor time over 20 s (they should use none).
2. The packaged licence check's parts, timed on this PC (development workers skip it):
   `studio_license_check hwid` (process start + machine ID v3), N at once, against a process that
   only starts; and SHA-256 of a file the size of the packaged studio_server.exe, N at once
   (the module signature check reads and hashes it).
Writes .local/trials/results/r3-cost-<time>.json.
"""
import argparse
import ctypes
from ctypes import wintypes
import hashlib
import json
import os
from pathlib import Path
import statistics
import subprocess
import sys
import time
from concurrent.futures import ThreadPoolExecutor

sys.path.insert(0, str(Path(__file__).resolve().parent))
import projects  # noqa: E402
from instance import Instance  # noqa: E402
from step0_run_stop import RESULTS  # noqa: E402
from winproc import cpu_seconds  # noqa: E402


class Counters(ctypes.Structure):
    _fields_ = [("cb", wintypes.DWORD), ("PageFaultCount", wintypes.DWORD)] + [
        (name, ctypes.c_size_t) for name in ("PeakWorkingSetSize", "WorkingSetSize", "QuotaPeakPagedPoolUsage",
                                             "QuotaPagedPoolUsage", "QuotaPeakNonPagedPoolUsage",
                                             "QuotaNonPagedPoolUsage", "PagefileUsage", "PeakPagefileUsage",
                                             "PrivateUsage")]


def private_mb(pid):
    kernel, psapi = ctypes.WinDLL("kernel32"), ctypes.WinDLL("psapi")
    kernel.OpenProcess.restype = ctypes.c_void_p
    handle = kernel.OpenProcess(0x1000, False, pid)
    counters = Counters()
    counters.cb = ctypes.sizeof(Counters)
    try:
        psapi.GetProcessMemoryInfo(ctypes.c_void_p(handle), ctypes.byref(counters), counters.cb)
    finally:
        kernel.CloseHandle(ctypes.c_void_p(handle))
    return counters.PrivateUsage / 2**20


def spares(server, native):
    with Instance(server=server, native=native, shared=True) as engine:
        projects.open_in(engine, projects.build("large"))
        want = len(engine.json("GET", "/api/v1/project")["ieds"])
        deadline, pids = time.time() + 60, []
        while time.time() < deadline:
            pids = [pid for pid, name in engine.job.processes().items() if name.lower() == "studio_ied.exe"]
            if len(pids) >= want:
                break
            time.sleep(0.2)
        time.sleep(2)
        memory = [private_mb(pid) for pid in pids]
        before = sum(cpu_seconds(pids).values())
        time.sleep(20)
        return {"spares": len(pids), "wanted": want, "private_mb_each": round(statistics.median(memory), 2),
                "private_mb_total": round(sum(memory), 1), "cpu_seconds_in_20s": round(sum(cpu_seconds(pids).values()) - before, 3)}


def timed_parallel(command, count):
    started = time.perf_counter()
    with ThreadPoolExecutor(count) as pool:
        list(pool.map(lambda _: subprocess.run(command, capture_output=True), range(count)))
    return time.perf_counter() - started


def hash_parallel(path, count):
    def one(_):
        digest = hashlib.sha256()
        with open(path, "rb") as stream:
            for block in iter(lambda: stream.read(1 << 20), b""):
                digest.update(block)
    started = time.perf_counter()
    with ThreadPoolExecutor(count) as pool:
        list(pool.map(one, range(count)))
    return time.perf_counter() - started


def licence(native, server, counts=(8, 20), repeats=5):
    check, bare = str(Path(native) / "studio_license_check.exe"), ["cmd", "/c", "exit"]
    result = {"module_mb": round(os.path.getsize(server) / 2**20, 1)}
    for count in counts:
        hwid = [timed_parallel([check, "hwid"], count) for _ in range(repeats)]
        start = [timed_parallel(bare, count) for _ in range(repeats)]
        hashed = [hash_parallel(server, count) for _ in range(repeats)]
        result[str(count)] = {"hwid_s": round(statistics.median(hwid), 3), "bare_start_s": round(statistics.median(start), 3),
                              "hash_s": round(statistics.median(hashed), 3)}
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--server", required=True)
    parser.add_argument("--native", required=True)
    parser.add_argument("--release", default=str(projects.REPO / "studio/native/build/Release"),
                        help="folder with studio_license_check.exe")
    args = parser.parse_args()
    result = {"spares": spares(args.server, args.native), "licence": licence(args.release, args.server)}
    RESULTS.mkdir(parents=True, exist_ok=True)
    path = RESULTS / f"r3-cost-{time.strftime('%Y%m%d-%H%M%S')}.json"
    path.write_text(json.dumps(result, indent=1), encoding="utf-8")
    print(json.dumps(result, indent=1), f"\nsaved {path}")


if __name__ == "__main__":
    main()
