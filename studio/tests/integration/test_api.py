from pathlib import Path
import io
import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]


@pytest.fixture
def app(tmp_path):
    settings = Settings(tmp_path, ROOT / "studio/native/build/Release/studio_ied.exe", tmp_path)
    application = create_app(settings, "test-session-token")
    yield application
    application.extensions["runtime"].close()


def test_auth_and_cross_origin_requests(app):
    client = app.test_client()
    missing = client.get("/api/v1/project")
    assert missing.status_code == 401
    assert "SubstationStudio.exe" in missing.json["error"] and "#session=" in missing.json["error"]
    assert client.post("/api/v1/session", json={"token": "test-session-token"}).status_code == 200
    assert client.get("/api/v1/project").status_code == 200
    response = client.post("/api/v1/ieds/template", json={"name": "Bad"}, headers={"Origin": "https://evil.example"})
    assert response.status_code == 403
    assert response.json["active_frontend"] is None


def test_dev_frontend_origin_is_exactly_scoped(tmp_path):
    settings = Settings(tmp_path, ROOT / "studio/native/build/Release/studio_ied.exe", tmp_path)
    application = create_app(settings, "dev-token", admin_origin="http://127.0.0.1:5174")
    try:
        client = application.test_client()
        missing = client.get("/api/v1/project", headers={"X-Studio-Frontend": "http://127.0.0.1:5174"})
        assert missing.status_code == 401 and "start-studio.cmd" in missing.json["error"]
        allowed = client.post("/api/v1/session", json={"token": "dev-token"},
                              headers={"Origin": "http://127.0.0.1:5174"})
        denied = client.post("/api/v1/session", json={"token": "dev-token"},
                             headers={"Origin": "http://127.0.0.1:5175"})
        stale_tab = client.post("/api/v1/session", json={"token": "dev-token"},
                                headers={"X-Studio-Frontend": "http://127.0.0.1:5173"})
        assert allowed.status_code == 200
        assert denied.status_code == 403
        assert denied.json["active_frontend"] == "http://127.0.0.1:5174"
        assert stale_tab.status_code == 403
    finally:
        application.extensions["runtime"].close()


def test_template_project_roundtrip(app):
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "test-session-token"})
    assert client.post("/api/v1/ieds/template", json={"name": "Feeder01"}).status_code == 200
    result = client.get("/api/v1/project").get_json()
    assert result["ieds"][0]["ip"] == "127.50.0.2"
    assert "cfg" not in result["ieds"][0]
    assert "source" not in result["ieds"][0]
    archive = client.get("/api/v1/project/download")
    assert archive.status_code == 200
    projects = app.extensions["projects"]
    projects.restore(archive.data)
    assert projects.snapshot()["ieds"][0]["name"] == "Feeder01"


def test_tabs_share_session_revision_and_stale_tab_cannot_overwrite(app):
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "test-session-token"})
    initial = client.get("/api/v1/project").json
    diagram = {"nodes": [{"id": "bus", "position": {"x": 0, "y": 0},
                          "data": {"label": "Bus", "kind": "BUSBAR", "rotation": 0}}],
               "edges": []}
    saved = client.put("/api/v1/project/diagram", json={
        "diagram": diagram, "revision": initial["revision"], "tab_id": "tab-a"})
    assert saved.status_code == 200 and saved.json["revision"] == initial["revision"] + 1
    current = client.get("/api/v1/project").json
    assert current["session_id"] == initial["session_id"]
    assert current["diagram"]["nodes"][0]["id"] == "bus"
    stale = client.put("/api/v1/project/diagram", json={
        "diagram": {"nodes": [], "edges": []},
        "revision": initial["revision"], "tab_id": "tab-b"})
    assert stale.status_code == 409
    assert client.get("/api/v1/project").json["diagram"]["nodes"][0]["id"] == "bus"
    changes = [event for event in app.extensions["events"].after(0)
               if event["kind"] == "project_changed"]
    assert changes[-1]["source"] == "tab-a"


def test_ied_deletion_requires_confirmation_and_stopped_runtime(app):
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "test-session-token"})
    key = app.extensions["projects"].add_template("DeleteMe")[0]["id"]
    assert client.delete(f"/api/v1/ieds/{key}").status_code == 400
    app.extensions["runtime"].state = "running"
    assert client.delete(f"/api/v1/ieds/{key}", json={"confirm": True}).status_code == 400
    app.extensions["runtime"].state = "stopped"
    assert client.get(f"/api/v1/ieds/{key}/removal-preview").json["name"] == "DeleteMe"
    assert client.delete(f"/api/v1/ieds/{key}", json={"confirm": True}).status_code == 200
    assert app.extensions["projects"].project["ieds"] == []


def test_csv_preview_and_project_roundtrip(app):
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "test-session-token"})
    projects = app.extensions["projects"]
    model = projects.add_template("Trace")[0]
    text = "seconds,voltage\n0,110\n1,115\n"
    preview = client.post("/api/v1/traces/preview", data={"file": (io.BytesIO(text.encode()), "trace.csv")})
    assert preview.status_code == 200 and preview.json["columns"] == ["seconds", "voltage"]
    spec = {"mode": "csv", "csv_text": text, "time_column": "seconds", "value_column": "voltage"}
    response = client.post("/api/v1/signals/source", json={"ied_id": model["id"],
                           "ref": "TraceCTRL/GGIO1.AnIn1.mag.f", "spec": spec})
    assert response.status_code == 200
    assert "csv_text" not in client.get("/api/v1/project").json["sources"][0]["spec"]
    projects.restore(projects.export())
    assert projects.project["sources"][0]["spec"]["csv_text"] == text
    assert projects.project["sources"][0]["spec"]["points"] == [[0, 110], [1, 115]]
