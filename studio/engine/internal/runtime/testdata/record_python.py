"""Run the scripted parity scenario on the frozen Python Coordinator and dump its events.

Usage: studio/.venv/Scripts/python.exe record_python.py NATIVE_DIR PORT_BASE OUT.json
Same steps as parityScript in script_test.go; IEDs of project.json move to PORT_BASE+1, +2.
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
from studio_app.simulation.equipment_test import apply_equipment_test  # noqa: E402


def main():
    native, port_base, out = Path(sys.argv[1]), int(sys.argv[2]), Path(sys.argv[3])
    project = json.loads((Path(__file__).parent / "project.json").read_text(encoding="utf-8"))
    for index, ied in enumerate(project["ieds"]):
        ied["port"] = port_base + index + 1
    ids = [ied["id"] for ied in project["ieds"]]
    with tempfile.TemporaryDirectory() as directory:
        projects = ProjectService(Path(directory), lambda revision, source: None)
        projects.project = project
        bus = EventBus(capacity=100000)
        c = Coordinator(projects, bus, native / "studio_ied.exe", Path(directory))
        timings = []
        steps = [
            ("start", lambda: c.start()),
            ("wait", lambda: time.sleep(0.5)),
            ("operate cb1 close", lambda: c.equipment_control.operate("cb1", value=True)),
            # the IED reports the new position at the worker's pace: let it arrive first
            ("wait", lambda: time.sleep(0.3)),
            ("operate ds1 open", lambda: c.equipment_control.operate("ds1", action="open")),
            ("wait", lambda: time.sleep(0.3)),
            ("manual AnIn2", lambda: c.update(ids[0], "BayACTRL/GGIO1.AnIn2.mag.f", 42.5)),
            ("pulse Loc", lambda: c.pulses.start(ids[0], "BayACTRL/XCBR1.Loc.stVal", True, 300)),
            ("wait", lambda: time.sleep(0.6)),
            ("scenario s1", lambda: c.start_scenario("s1")),
            ("wait", lambda: time.sleep(0.9)),
            ("test es1 closed", lambda: apply_equipment_test(c, "es1", "closed")),
            ("press pb1", lambda: c.equipment_control.operate("pb1", action="press")),
            ("stop BayB", lambda: c.stop([ids[1]])),
            ("start BayB", lambda: c.start([ids[1]])),
            ("wait", lambda: time.sleep(0.3)),
            ("stop", lambda: c.stop()),
        ]
        try:
            for name, action in steps:
                started = time.perf_counter()
                action()
                timings.append([name, time.perf_counter() - started])
        finally:
            c.close()
        events = bus.after(0)
    out.write_text(json.dumps({"events": events, "timings": timings}), encoding="utf-8")


if __name__ == "__main__":
    main()
