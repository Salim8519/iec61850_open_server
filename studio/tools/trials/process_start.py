"""Bare IED process start: spawn N studio_ied.exe at once and time each until its first answer.

    python studio/tools/trials/process_start.py --count 8 --repeats 10

The engine starts one studio_ied process per IED and sends "start" at once (native/worker.go);
the worker answers only after its own start-up, the licence check and the model load. Here each
process gets an unknown operation instead, which it refuses as soon as it can read requests, so
the time is the process start alone (no licence check, no model). Run it with the same count as
the project's IEDs to compare with the "model load" step of step0_run_stop.py.
"""
import argparse
import json
from pathlib import Path
from statistics import median
import struct
import subprocess
import sys
import threading
import time

sys.path.insert(0, str(Path(__file__).resolve().parent))
from instance import NATIVE  # noqa: E402

REQUEST = json.dumps({"version": 1, "id": "trial", "op": "trial_noop"}).encode()


def one(executable, results, index, barrier):
    barrier.wait()
    started = time.perf_counter()
    process = subprocess.Popen([str(executable)], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                               stderr=subprocess.DEVNULL, creationflags=subprocess.CREATE_NO_WINDOW)
    try:
        process.stdin.write(struct.pack("<I", len(REQUEST)) + REQUEST)
        process.stdin.flush()
        header = process.stdout.read(4)
        answer = process.stdout.read(struct.unpack("<I", header)[0])
        results[index] = time.perf_counter() - started
        if b"Unknown operation" not in answer:
            raise RuntimeError(f"unexpected answer: {answer[:200]!r}")
    finally:
        process.stdin.close()
        process.wait(10)


def round_once(executable, count):
    results, barrier = [None] * count, threading.Barrier(count)
    threads = [threading.Thread(target=one, args=(executable, results, i, barrier)) for i in range(count)]
    for thread in threads:
        thread.start()
    for thread in threads:
        thread.join()
    return results


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--count", type=int, default=8)
    parser.add_argument("--repeats", type=int, default=10)
    parser.add_argument("--exe", type=Path, default=NATIVE / "studio_ied.exe")
    args = parser.parse_args()
    slowest, typical = [], []
    for repeat in range(args.repeats):
        times = round_once(args.exe, args.count)
        slowest.append(max(times))
        typical.append(median(times))
        time.sleep(0.5)
    print(json.dumps({"count": args.count, "repeats": args.repeats, "exe": str(args.exe),
                      "slowest_process_median_s": round(median(slowest), 3), "slowest_process_worst_s": round(max(slowest), 3),
                      "typical_process_median_s": round(median(typical), 3)}, indent=1))


if __name__ == "__main__":
    main()
