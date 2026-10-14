"""Hints never block, and external switching against a hint is noted (spec interlock-hints, HINT-04)."""
import os
import subprocess
import time
from pathlib import Path
import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_ied.exe").exists(), reason="Native build required")


def node(node_id, kind, **data):
    return {"id": node_id, "position": {"x": 0, "y": 0}, "data": {"label": node_id.upper(), "kind": kind, **data}}


def wire(a, b):
    return {"id": f"{a}-{b}", "source": a, "target": b, "sourceHandle": "south", "targetHandle": "north"}


def observed(events):
    return [event for event in events.after(0) if event["kind"] == "interlock_hint_observed"]


def test_hints_never_block_and_external_switching_is_noted(tmp_path):
    app = create_app(Settings(tmp_path, BIN / "studio_ied.exe", tmp_path), "hint-token")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "hint-token"})
    projects = app.extensions["projects"]
    ied = projects.add_template("Mapped")[0]
    ied.update(ip="127.50.0.225", port=18115)
    status = "MappedCTRL/XSWI1.Pos.stVal"
    disconnector = node("q1", "DS", state_mapping={"mode": "direct", "source": {"ied_id": ied["id"], "ref": status, "report_ref": None}},
                        control_binding={"ied_id": ied["id"], "ref": "MappedCTRL/XCBR1.Pos"})
    # Q0 has no position signal: its position is unknown, so switching Q1 carries a hint.
    projects.diagram({"nodes": [node("feed", "INFEED"), disconnector, node("q0", "CB"), node("line", "JUNCTION")],
                      "edges": [wire("feed", "q1"), wire("q1", "q0"), wire("q0", "line")]})
    runtime, events = app.extensions["runtime"], app.extensions["events"]
    try:
        runtime.start()
        hints = client.get("/api/v1/equipment/states").json["energization"]["hints"]
        assert [hint["code"] for hint in hints["q1"]["close"]] == ["unknown"]
        # Studio's own Operate goes ahead (a hint never blocks) and is not reported as external.
        assert client.post("/api/v1/equipment/q1/operate", json={"value": True}).status_code == 200
        time.sleep(0.5)
        assert observed(events) == []
        # An independent MMS client opens Q1 against the hint: accepted, then noted in Activity.
        result = subprocess.run([str(BIN / "studio_probe.exe"), ied["ip"], str(ied["port"]), "control",
                                 "MappedCTRL/XCBR1.Pos", "false"], capture_output=True, text=True, timeout=15)
        assert result.returncode == 0, result.stdout + result.stderr
        for _ in range(60):
            if observed(events):
                break
            time.sleep(0.05)
        notes = observed(events)
        assert len(notes) == 1 and notes[0]["node_id"] == "q1" and notes[0]["action"] == "open"
        assert "opened Q1" in notes[0]["message"] and "unknown" in notes[0]["message"]
        assert runtime.supervisor.read(ied["id"], status) == 1, "the external command was not blocked"
    finally:
        runtime.close()
