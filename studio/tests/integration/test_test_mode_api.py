"""Test mode settings of simulated IEDs (spec TMODE)."""
import io
import json
from pathlib import Path

import pytest
from studio_app.app_factory import create_app
from studio_app.projects.archive import decode
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]
SOURCE = (ROOT / "scd/model_substation.scd").read_bytes()


@pytest.fixture
def client(tmp_path):
    app = create_app(Settings(tmp_path, tmp_path / "unused.exe", tmp_path), "mode-test")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "mode-test"})
    client.post("/api/v1/scl/import", data={"file": (io.BytesIO(SOURCE), "station.scd"),
                                            "selected_names": json.dumps(["FEED1", "BUS1"])})
    yield app, client
    app.extensions["runtime"].close()


def ids(app):
    return {ied["name"]: ied["id"] for ied in app.extensions["projects"].project["ieds"]}


def test_real_ied_is_refused_and_settings_persist(client):
    app, http = client
    feed, bus = ids(app)["FEED1"], ids(app)["BUS1"]
    saved = http.put(f"/api/v1/ieds/{feed}/test-mode", json={"devices": {"FEED1LD1": 3}, "simulation": True})
    assert saved.status_code == 200 and saved.json["written"] == 0
    # A later role or GOOSE edit keeps the test settings.
    assert http.put(f"/api/v1/ieds/{feed}/settings", json={"goose_publish": False}).status_code == 200
    stored = decode(app.extensions["projects"].export())
    feed_stored = next(item for item in stored["ieds"] if item["name"] == "FEED1")
    assert feed_stored["test_mode"] == {"devices": {"FEED1LD1": 3}, "simulation": True}
    health = http.get("/api/v1/health").json
    assert health["modes"]["ieds"][feed]["devices"]["FEED1LD1"] == "test"
    http.put(f"/api/v1/ieds/{bus}/settings", json={"role": "real"})
    refused = http.put(f"/api/v1/ieds/{bus}/test-mode", json={"devices": {"BUS1LD1": 5}})
    assert refused.status_code == 400 and "never changes a real device" in refused.json["error"]
    assert http.put(f"/api/v1/ieds/{feed}/test-mode", json={"devices": {"FEED1LD1": 7}}).status_code == 400
