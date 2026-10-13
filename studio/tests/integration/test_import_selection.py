"""Import requires explicit IED selection; preview must never mutate a project."""
import io
import json
from pathlib import Path

import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings
from studio_app.projects.archive import decode

ROOT = Path(__file__).resolve().parents[3]
SOURCE = (ROOT / "scd/open_substation.scd").read_bytes()


@pytest.fixture
def session(tmp_path):
    app = create_app(Settings(tmp_path, tmp_path / "unused.exe", tmp_path), "selection-test")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "selection-test"})
    yield app, client
    app.extensions["runtime"].close()


def upload(client, path, names=None):
    data = {"file": (io.BytesIO(SOURCE), "station.cid")}
    if names is not None:
        data["selected_names"] = json.dumps(names)
    return client.post("/api/v1/scl/" + path, data=data)


def test_preview_and_select_one_ied(session):
    app, client = session
    before = decode(app.extensions["projects"].export())
    response = upload(client, "preview")
    assert response.status_code == 200
    assert len(response.json["ieds"]) == 4
    assert decode(app.extensions["projects"].export()) == before
    name = response.json["ieds"][0]["name"]
    assert upload(client, "import", [name]).status_code == 200
    projects = app.extensions["projects"]
    assert [ied["name"] for ied in projects.project["ieds"]] == [name]
    assert projects.project["ieds"][0]["source"].encode() == SOURCE
    projects.restore(projects.export())
    assert [ied["name"] for ied in projects.project["ieds"]] == [name]
    again = upload(client, "preview").json
    assert next(ied for ied in again["ieds"] if ied["name"] == name)["unavailable_reason"]


@pytest.mark.parametrize("selection", [None, [], ["not-in-file"], ["IED1_XCBR", "IED1_XCBR"]])
def test_missing_or_invalid_selection_does_not_import(session, selection):
    app, client = session
    assert upload(client, "import", selection).status_code == 400
    assert app.extensions["projects"].project["ieds"] == []
