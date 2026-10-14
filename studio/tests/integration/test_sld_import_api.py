"""SLD import preview endpoint (spec SLD)."""
import io
import json
from pathlib import Path

import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]
SOURCE = (ROOT / "scd/open_substation.scd").read_bytes()


@pytest.fixture
def client(tmp_path):
    app = create_app(Settings(tmp_path, tmp_path / "unused.exe", tmp_path), "sld-test")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "sld-test"})
    client.post("/api/v1/scl/import", data={"file": (io.BytesIO(SOURCE), "station.scd"),
                                            "selected_names": json.dumps(["IED1_XCBR"])})
    yield app, client
    app.extensions["runtime"].close()


def source_id(client):
    sources = client.get("/api/v1/project/substation-sources").json["sources"]
    assert sources[0]["substations"] == ["S12"]
    return sources[0]["id"]


def test_preview_does_not_change_project(client):
    app, http = client
    service = app.extensions["projects"]
    before, revision = service.export(), service.revision
    response = http.post("/api/v1/project/diagram/from-scl/preview",
                         json={"source_id": source_id(http), "mode": "replace"})
    assert response.status_code == 200, response.json
    assert service.export() == before and service.revision == revision
    assert response.json["revision"] == revision


def test_preview_binds_lnodes_and_applies(client):
    app, http = client
    preview = http.post("/api/v1/project/diagram/from-scl/preview",
                        json={"source_id": source_id(http), "mode": "replace"}).json
    breaker = next(n for n in preview["diagram"]["nodes"] if n["data"]["label"] == "QA1")
    assert breaker["data"]["state_mapping"]["source"]["ref"] == "IED1_XCBRGenericIO/XCBR1.Pos.stVal"
    assert any(item["path"].endswith("QA1") for item in preview["unbound"])
    assert {item["type"] for item in preview["skipped"]} == {"IFL", "PowerTransformer"}
    saved = http.put("/api/v1/project/diagram", json={"diagram": preview["diagram"],
                                                      "revision": preview["revision"], "tab_id": "t1"})
    assert saved.status_code == 200
    assert len(app.extensions["projects"].project["diagram"]["nodes"]) == len(preview["diagram"]["nodes"])


def test_unknown_source_and_mode_are_refused(client):
    _, http = client
    assert http.post("/api/v1/project/diagram/from-scl/preview",
                     json={"source_id": "nope", "mode": "replace"}).status_code == 400
    assert http.post("/api/v1/project/diagram/from-scl/preview",
                     json={"source_id": source_id(http), "mode": "wipe"}).status_code == 400
