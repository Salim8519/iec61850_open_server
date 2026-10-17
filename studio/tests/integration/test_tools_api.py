"""Tools API: IED Explorer (with confirmed commands) and Find IEDs over HTTP, against a studio_ied stand-in."""
import os
import time
from pathlib import Path

import pytest
from studio_app.app_factory import create_app
from studio_app.runtime.worker_process import WorkerProcess
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
IP, PORT = "127.50.0.236", 18136


@pytest.fixture
def client(tmp_path):
    settings = Settings(tmp_path, BIN / "studio_ied.exe", tmp_path)
    app = create_app(settings, "tools-token")
    assert app.extensions["startup"].wait(30)
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "tools-token"})
    yield client
    app.extensions["explorer"].disconnect()
    app.extensions["explorer"].close()
    app.extensions["explorer"].control.close()
    app.extensions["scanner"].stop()
    app.extensions["runtime"].close()


def test_bad_input_gets_actionable_errors(client):
    assert "four numbers" in client.post("/api/v1/tools/explorer/connect", json={"host": "x"}).json["error"]
    assert "Connect to a device first" in client.get("/api/v1/tools/explorer/devices").json["error"]
    assert "at most 256" in client.post("/api/v1/tools/scan/preview", json={"range": "10.0.0.0/8"}).json["error"]
    assert client.post("/api/v1/tools/explorer/read", json={"items": "x"}).status_code == 400
    preview = client.post("/api/v1/tools/scan/preview", json={"range": "10.0.0.0/24"}).json
    assert preview["count"] == 254 and preview["port"] == 102
    assert client.get("/api/v1/tools/scan").json["state"] == "idle"
    refused = client.post("/api/v1/tools/explorer/operate", json={"ref": "LD0/XCBR1.Pos", "value": True})
    assert refused.status_code == 400 and "Confirm the command first" in refused.json["error"]


@pytest.mark.skipif(not (BIN / "studio_live.exe").exists(), reason="Native build required")
def test_explorer_and_find_ieds_over_http(client, tmp_path):
    model = import_scl(make_template("ApiTools"))[0]
    config = tmp_path / "device.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    try:
        worker.request("start", cfg=str(config), ip=IP, port=PORT, controls=model["controls"], edition=2)
        assert client.post("/api/v1/tools/explorer/connect", json={"host": IP, "port": PORT}).json["state"] == "connected"
        assert "ApiToolsCTRL" in client.get("/api/v1/tools/explorer/devices").json["names"]
        assert "XCBR1" in client.get("/api/v1/tools/explorer/nodes?device=ApiToolsCTRL").json["names"]
        node = client.get("/api/v1/tools/explorer/node?ref=ApiToolsCTRL/XCBR1").json
        pos = next(obj for obj in node["objects"] if obj["name"] == "Pos")
        read = client.post("/api/v1/tools/explorer/read", json={"items": pos["attributes"][:3]}).json["values"]
        assert read and all("text" in value for value in read.values())
        watch = client.put("/api/v1/tools/explorer/watch", json={"items": pos["attributes"][:2], "interval_ms": 500}).json
        assert len(watch["watch"]) == 2
        assert client.post("/api/v1/tools/explorer/polling", json={"on": True}).json["polling"] is True
        time.sleep(1.2)
        csv_text = client.get("/api/v1/tools/explorer/watch.csv").get_data(as_text=True)
        assert csv_text.startswith("reference,fc,value") and "ApiToolsCTRL/XCBR1.Pos" in csv_text
        answer = client.post("/api/v1/tools/explorer/operate",
                             json={"ref": "ApiToolsCTRL/XCBR1.Pos", "value": True, "confirm": True}).json
        assert answer["result"]["ok"] is True and answer["observed"]["text"] == "on (closed)"
        remembered = client.get("/api/v1/tools/memory").json["explorer"]
        assert remembered["host"] == IP and remembered["watch_by_host"][f"{IP}:{PORT}"]
        assert client.post("/api/v1/tools/explorer/disconnect").json["state"] == "disconnected"
        client.post("/api/v1/tools/scan/start", json={"range": f"{IP}-237", "port": PORT})
        deadline = time.time() + 20
        while time.time() < deadline and client.get("/api/v1/tools/scan").json["state"] == "scanning":
            time.sleep(0.05)
        status = client.get("/api/v1/tools/scan").json
        assert status["state"] == "done" and status["found"][0]["identified"] is True
    finally:
        worker.close()
