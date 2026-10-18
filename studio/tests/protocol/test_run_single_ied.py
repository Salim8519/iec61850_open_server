"""Run or stop one IED while another keeps running; an independent client sees only running IEDs.

Also checks that an equipment test button changes the simulated position over MMS
without sending any command (ADR-0067, change 0097).
"""
import json
import os
import subprocess
from pathlib import Path
import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_probe.exe").exists(), reason="Native build required")
A = ("127.50.0.231", 18131)
B = ("127.50.0.232", 18132)


def probe(endpoint, reference):
    result = subprocess.run([str(BIN / "studio_probe.exe"), endpoint[0], str(endpoint[1]), "read", reference, "ST"],
                            capture_output=True, text=True, timeout=20)
    try:
        return json.loads(result.stdout).get("value") if result.returncode == 0 else None
    except json.JSONDecodeError:
        return None


@pytest.fixture
def app(tmp_path):
    app = create_app(Settings(tmp_path, BIN / "studio_ied.exe", tmp_path), "run-one")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "run-one"})
    projects = app.extensions["projects"]
    first, second = projects.add_template("RunA")[0], projects.add_template("RunB")[0]
    first.update(ip=A[0], port=A[1])
    second.update(ip=B[0], port=B[1])
    status = "RunACTRL/XSWI1.Pos.stVal"
    projects.diagram({"nodes": [{"id": "cb", "position": {"x": 0, "y": 0}, "data": {
        "label": "Q0", "kind": "CB", "state_mapping": {"mode": "direct", "source": {
            "ied_id": first["id"], "ref": status, "report_ref": None}}}}], "edges": []})
    yield app, client, first, second, status
    app.extensions["runtime"].close()


def test_one_ied_runs_alone_then_joins_and_leaves(app):
    app, client, first, second, status = app
    runtime = app.extensions["runtime"]
    assert client.post(f"/api/v1/runtime/ieds/{first['id']}/start").json["running"] == [first["id"]]
    assert probe(A, status) is not None, "the running IED answers an independent client"
    assert probe(B, "RunBCTRL/XSWI1.Pos.stVal") is None, "the other IED is not running"
    assert client.get("/api/v1/project").json["running_ieds"] == [first["id"]]
    client.post(f"/api/v1/runtime/ieds/{second['id']}/start")
    assert runtime.active == {first["id"], second["id"]}
    assert probe(B, "RunBCTRL/XSWI1.Pos.stVal") is not None
    client.post(f"/api/v1/runtime/ieds/{first['id']}/stop")
    assert runtime.state == "running" and runtime.active == {second["id"]}
    assert probe(A, status) is None and first["state"] == "stopped"
    assert probe(B, "RunBCTRL/XSWI1.Pos.stVal") is not None, "stopping one IED keeps the other"
    states = client.get("/api/v1/equipment/states").json["states"]
    assert states == [{"node_id": "cb", "value": None}], "equipment of a stopped IED has no value"
    client.post(f"/api/v1/runtime/ieds/{second['id']}/stop")
    assert runtime.state == "stopped" and not runtime.supervisor.workers


def test_test_button_moves_breaker_without_command(app):
    app, client, first, _, status = app
    client.post(f"/api/v1/runtime/ieds/{first['id']}/start")
    for action, expected in (("closed", "10"), ("moving", "00"), ("bad", "11"), ("open", "01")):
        response = client.post("/api/v1/equipment/cb/test", json={"action": action})
        assert response.status_code == 200, response.json
        assert probe(A, status) == expected
    events = app.extensions["events"].after(0)
    assert not [event for event in events if event["kind"] == "control_applied"], "no command is sent"
    first["role"] = "real"
    refused = client.post("/api/v1/equipment/cb/test", json={"action": "closed"})
    assert refused.status_code == 400 and "real device" in refused.json["error"]
