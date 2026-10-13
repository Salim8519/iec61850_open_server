import time
import os
from pathlib import Path

import pytest
from studio_app.projects.service import ProjectService
from studio_app.runtime.event_bus import EventBus
from studio_app.simulation.coordinator import Coordinator

ROOT = Path(__file__).resolve().parents[3]
WORKER = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release")) / "studio_ied.exe"
pytestmark = pytest.mark.skipif(not WORKER.exists(), reason="Native build required")


def test_scenario_pause_resume_and_native_assertion(tmp_path):
    projects = ProjectService(tmp_path)
    ied = projects.add_template("Timeline")[0]
    ied.update(ip="127.50.0.219", port=18106)
    reference = "TimelineCTRL/GGIO1.AnIn1.mag.f"
    projects.scenarios([{"id": "sequence", "name": "Sequence", "enabled": True, "steps": [
        {"id": "set", "at": 0.6, "ied_id": ied["id"], "ref": reference, "value": 42.5},
        {"id": "check", "at": 0.8, "action": "assert", "ied_id": ied["id"], "ref": reference, "value": 42.5}]}])
    projects.diagram({"nodes": [{"id": "cb", "position": {"x": 0, "y": 0},
                      "data": {"label": "CB", "kind": "CB", "state_mapping": {"mode": "direct",
                               "source": {"ied_id": ied["id"],
                                          "ref": "TimelineCTRL/XCBR1.Pos.stVal"}}}}], "edges": []})
    bus = EventBus()
    runtime = Coordinator(projects, bus, WORKER, tmp_path)
    try:
        runtime.start()
        assert any(event.get("source") == "initial_native_snapshot" and event["value"] == 1
                   for event in bus.after(0))
        assert any(event["kind"] == "equipment_state" and event["node_id"] == "cb"
                   and event["value"] == 1 for event in bus.after(0))
        runtime.pause()
        time.sleep(0.9)
        assert runtime.supervisor.read(ied["id"], reference) == 0
        runtime.pause()
        deadline = time.monotonic() + 3
        while runtime.scenario_runner.snapshot()[0]["state"] == "running" and time.monotonic() < deadline:
            time.sleep(0.05)
        assert runtime.scenario_runner.snapshot()[0]["state"] == "completed"
        assert runtime.supervisor.read(ied["id"], reference) == 42.5
    finally:
        runtime.close()
