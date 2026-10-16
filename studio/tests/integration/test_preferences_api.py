"""Guided-tour state lives in the data folder, so it survives port changes and restarts."""
from pathlib import Path
from studio_app.app_factory import create_app
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]


def client_for(tmp_path):
    settings = Settings(tmp_path, ROOT / "studio/native/build/Release/studio_ied.exe", tmp_path)
    app = create_app(settings, "prefs-token")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "prefs-token"})
    return app, client


def test_tour_is_due_on_first_start_and_remembered_after_finishing(tmp_path):
    app, client = client_for(tmp_path)
    try:
        first = client.get("/api/v1/preferences").json
        assert first["tour_seen_version"] < first["tour_version"]
        saved = client.put("/api/v1/preferences", json={"tour_seen_version": first["tour_version"]})
        assert saved.status_code == 200 and saved.json["tour_seen_version"] == first["tour_version"]
    finally:
        app.extensions["runtime"].close()
    app, client = client_for(tmp_path)
    try:
        assert client.get("/api/v1/preferences").json["tour_seen_version"] == first["tour_version"]
    finally:
        app.extensions["runtime"].close()


def test_unknown_or_wrongly_typed_preferences_are_refused(tmp_path):
    app, client = client_for(tmp_path)
    try:
        assert client.put("/api/v1/preferences", json={"admin": True}).status_code == 400
        assert client.put("/api/v1/preferences", json={"tour_seen_version": "yes"}).status_code == 400
        assert client.put("/api/v1/preferences", json={"tour_seen_version": True}).status_code == 400
        assert client.put("/api/v1/preferences", json=[1]).status_code == 400
    finally:
        app.extensions["runtime"].close()
