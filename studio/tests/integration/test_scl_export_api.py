"""Verified SCL API downloads are authenticated, explicit, and non-mutating."""
import pytest
from studio_app.app_factory import create_app
from studio_app.projects.archive import decode
from studio_app.scl.importer import import_scl
from studio_app.settings import Settings


@pytest.fixture
def session(tmp_path):
    app = create_app(Settings(tmp_path, tmp_path / "unused.exe", tmp_path), "export-test")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "export-test"})
    ied = app.extensions["projects"].add_template("ExportBay")[0]
    yield app, client, ied
    app.extensions["runtime"].close()


def test_download_is_verified_and_does_not_change_project(session):
    app, client, ied = session
    before = decode(app.extensions["projects"].export())
    point = next(item for item in ied["signals"] if item["kind"] == "FLOAT32")
    response = client.post(f"/api/v1/ieds/{ied['id']}/export-scl",
                           json={"overrides": {point["ref"]: 37.25}})
    assert response.status_code == 200
    assert "ExportBay.edited.scd" in response.headers["Content-Disposition"]
    model = import_scl(response.data)[0]
    assert next(item for item in model["signals"] if item["ref"] == point["ref"])["value"] == 37.25
    assert decode(app.extensions["projects"].export()) == before
    assert client.get(f"/api/v1/ieds/{ied['id']}/source").data == ied["source"].encode()
    anonymous = app.test_client()
    assert anonymous.post(f"/api/v1/ieds/{ied['id']}/export-scl", json={"overrides": {}}).status_code == 401


@pytest.mark.parametrize("body", [{}, {"overrides": []}, {"overrides": {}, "simulation_ips": True}])
def test_invalid_export_options_are_rejected(session, body):
    _, client, ied = session
    assert client.post(f"/api/v1/ieds/{ied['id']}/export-scl", json=body).status_code == 400
