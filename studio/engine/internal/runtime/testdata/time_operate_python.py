"""Time canvas CB operations on the frozen Python Coordinator (synthetic template project).

Usage: studio/.venv/Scripts/python.exe time_operate_python.py NATIVE_DIR PORT_BASE COUNT
Prints JSON {"done_ms": [...], "symbol_ms": [...]}: "done" is the operate call, "symbol" the
first equipment_state event of the node with the new position after the call started.
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
    native, port_base, count = Path(sys.argv[1]), int(sys.argv[2]), int(sys.argv[3])
    project = json.loads((Path(__file__).parent / "project.json").read_text(encoding="utf-8"))
    project["sources"] = []  # no generator traffic while timing
    for index, ied in enumerate(project["ieds"]):
        ied["port"] = port_base + index + 1
    done, symbol = [], []
    with tempfile.TemporaryDirectory() as directory:
        projects = ProjectService(Path(directory), lambda revision, source: None)
        projects.project = project
        bus = EventBus(capacity=100000)
        c = Coordinator(projects, bus, native / "studio_ied.exe", Path(directory))
        try:
            c.start()
            time.sleep(0.5)
            for index in range(count):
                close = index % 2 == 0
                cursor = bus.sequence
                started = time.time()
                c.equipment_control.operate("cb1", value=close)
                done.append((time.time() - started) * 1000)
                wanted = 2 if close else 1
                deadline = time.time() + 3
                while time.time() < deadline:
                    hits = [event for event in bus.after(cursor) if event["kind"] == "equipment_state"
                            and event["node_id"] == "cb1" and event["value"] == wanted]
                    if hits:
                        symbol.append((hits[0]["timestamp"] - started) * 1000)
                        break
                    time.sleep(0.005)
                time.sleep(0.1)
            c.stop()
        finally:
            c.close()
    print(json.dumps({"done_ms": done, "symbol_ms": symbol}))


if __name__ == "__main__":
    main()
