"""Authenticated HTTP workflows for report filtering, panels and atomic deletion."""
from pathlib import Path
import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings


@pytest.fixture
def lab(tmp_path):
    app = create_app(Settings(tmp_path, Path("unused-worker"), tmp_path), "mapping-token")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "mapping-token"})
    projects = app.extensions["projects"]
    model = projects.add_template("Mapping")[0]
    try:
        yield client, app, projects, model
    finally:
        app.extensions["runtime"].close()


def test_filtered_search_and_saved_panel(lab):
    client, _, projects, model = lab
    key = model["id"]
    response = client.get(f"/api/v1/ieds/{key}/signals", query_string={"report": "MappingCTRL/LLN0.BR.Buffered"})
    assert response.status_code == 200 and response.json["total"] == 5
    assert client.get(f"/api/v1/ieds/{key}/signals", query_string={"report": "wrong"}).status_code == 400
    model["signals"][0]["description"] = "Unique descriptive name"
    found = client.get(f"/api/v1/ieds/{key}/signals", query_string={"q": "unique descriptive"})
    assert found.json["total"] == 1
    boolean = client.get(f"/api/v1/ieds/{key}/signals", query_string={
        "kind": "BOOLEAN", "report": "MappingCTRL/LLN0.BR.Buffered"})
    assert boolean.status_code == 200 and boolean.json["total"] == 1
    editable = client.get(f"/api/v1/ieds/{key}/signals", query_string={"writable": "true"})
    assert editable.status_code == 200
    assert editable.json["items"]
    assert all(item["writable"] and item["fc"] != "CF" for item in editable.json["items"])
    entry = {"id": "saved", "label": "Alarm", "ied_id": key, "ref": "MappingCTRL/XCBR1.Loc.stVal"}
    assert client.put("/api/v1/project/test-panel", json=[entry]).status_code == 200
    assert client.get("/api/v1/project/test-panel").json["items"][0]["signal"]["kind"] == "BOOLEAN"
    assert not projects.project["diagram"]["nodes"]


def test_bulk_confirmation_stale_preview_and_running_guard(lab):
    client, app, projects, model = lab
    ids = [model["id"], projects.add_template("Other")[0]["id"]]
    preview = client.post("/api/v1/ieds/removal-preview", json={"ids": ids}).json
    body = {"ids": ids, "fingerprint": preview["fingerprint"]}
    assert client.post("/api/v1/ieds/remove", json=body).status_code == 400
    projects.project["name"] = "Changed since preview"
    assert client.post("/api/v1/ieds/remove", json={**body, "confirm": True}).status_code == 400
    preview = client.post("/api/v1/ieds/removal-preview", json={"ids": ids}).json
    body.update(confirm=True, fingerprint=preview["fingerprint"])
    app.extensions["runtime"].state = "running"
    assert client.post("/api/v1/ieds/remove", json=body).status_code == 400
    assert client.put("/api/v1/project/test-panel", json=[]).status_code == 400
    app.extensions["runtime"].state = "stopped"
    assert client.post("/api/v1/ieds/remove", json=body).status_code == 200
    assert projects.project["ieds"] == []
