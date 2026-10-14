"""Live colouring follows real switching over native MMS (spec energization, POWER-05).

Energisation only reads positions; `energization_states.py` has no write path at all.
"""
import os
from pathlib import Path
import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_ied.exe").exists(), reason="Native build required")


def node(node_id, kind, **data):
    return {"id": node_id, "position": {"x": 0, "y": 0}, "data": {"label": node_id, "kind": kind, **data}}


def test_energization_events_follow_switching(tmp_path):
    app = create_app(Settings(tmp_path, BIN / "studio_ied.exe", tmp_path), "live-token")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "live-token"})
    projects = app.extensions["projects"]
    ied = projects.add_template("Mapped")[0]
    ied.update(ip="127.50.0.224", port=18114)
    status = "MappedCTRL/XSWI1.Pos.stVal"
    breaker = node("cb", "CB", state_mapping={"mode": "direct", "source": {"ied_id": ied["id"], "ref": status, "report_ref": None}},
                   control_binding={"ied_id": ied["id"], "ref": "MappedCTRL/XCBR1.Pos"})
    wires = [{"id": "in", "source": "feed", "target": "cb", "sourceHandle": "south", "targetHandle": "north"},
             {"id": "out", "source": "cb", "target": "load", "sourceHandle": "south", "targetHandle": "north"}]
    projects.diagram({"nodes": [node("feed", "INFEED"), breaker, node("load", "JUNCTION")], "edges": wires})
    runtime = app.extensions["runtime"]
    events = app.extensions["events"]
    stopped = client.get("/api/v1/equipment/states").json["energization"]
    assert stopped == {"nodes": {}, "edges": {}, "hints": {}}, "nothing is coloured while stopped"
    try:
        runtime.start()
        assert client.post("/api/v1/equipment/cb/operate", json={"value": True}).status_code == 200
        assert client.get("/api/v1/equipment/states").json["energization"]["edges"] == {"in": "live", "out": "live"}
        assert client.post("/api/v1/equipment/cb/operate", json={"value": False}).status_code == 200
        published = [event for event in events.after(0) if event["kind"] == "energization"]
        assert published[-1]["edges"] == {"in": "live", "out": "dead"}
        assert published[-1]["nodes"]["cb"] == ["live", "dead"]
        outs = [event["edges"]["out"] for event in published]
        assert all(a != b for a, b in zip(outs, outs[1:])), "published only when something changed"
        # Energisation only reads: the IED value is whatever the operation left, never rewritten.
        assert runtime.supervisor.read(ied["id"], status) == 1
    finally:
        runtime.close()
