"""Time Run and Stop of the demo station on the frozen Python Coordinator (lane I2, gap 2).

Usage: studio/.venv/Scripts/python.exe -I stop_timing_python.py NATIVE_DIR SCD HOST_PREFIX PORT_BASE ROUNDS OUT.json

Imports every IED of SCD, moves IED i to HOST_PREFIX.(11+i) port PORT_BASE+1+i (loopback,
127.67.x.y), runs ROUNDS cycles of start, 0.5 s wait, stop and reports the seconds of each
start and stop (coordinator calls, no HTTP), like TestComposedStopTiming does in Go.
"""
import json
import sys
import tempfile
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[5]
sys.path.insert(0, str(ROOT / "studio" / "backend"))

from studio_app.projects.service import ProjectService  # noqa: E402
from studio_app.runtime.event_bus import EventBus  # noqa: E402
from studio_app.simulation.coordinator import Coordinator  # noqa: E402


def main():
    native, scd, prefix, port_base = Path(sys.argv[1]), Path(sys.argv[2]), sys.argv[3], int(sys.argv[4])
    rounds, out = int(sys.argv[5]), Path(sys.argv[6])
    starts, stops = [], []
    with tempfile.TemporaryDirectory() as directory:
        projects = ProjectService(Path(directory), lambda revision, source: None)
        projects.add_scl(scd.read_text(encoding="utf-8"), scd.name)
        for index, ied in enumerate(projects.project["ieds"]):
            ied.update(ip=f"{prefix}.{11 + index}", port=port_base + 1 + index)
        c = Coordinator(projects, EventBus(capacity=100000), native / "studio_ied.exe", Path(directory))
        try:
            for _ in range(rounds):
                started = time.perf_counter()
                c.start()
                starts.append(time.perf_counter() - started)
                time.sleep(0.5)
                started = time.perf_counter()
                c.stop()
                stops.append(time.perf_counter() - started)
                time.sleep(0.2)
        finally:
            c.close()
    out.write_text(json.dumps({"ieds": len(projects.project["ieds"]), "start": starts, "stop": stops}),
                   encoding="utf-8")


if __name__ == "__main__":
    main()
