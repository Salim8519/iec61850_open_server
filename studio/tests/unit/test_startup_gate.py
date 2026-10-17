"""The window opens before the autosave is restored; engineering requests wait for it."""
import threading

from studio_app.app_factory import create_app
from studio_app.projects import startup_recovery
from studio_app.projects.service import ProjectService
from studio_app.settings import Settings


class UnlicensedService:
    enforce = True

    def is_licensed(self):
        return False

    def start_background_verify(self):
        pass


def make_app(tmp_path, **options):
    return create_app(Settings(tmp_path, tmp_path / "missing.exe", tmp_path), "gate-test", **options)


def signed_in(app):
    client = app.test_client()
    assert client.post("/api/v1/session", json={"token": "gate-test"}).status_code == 200
    return client


def hold_recovery(monkeypatch):
    release = threading.Event()
    original = startup_recovery.recover_startup

    def held(projects):
        release.wait(30)
        return original(projects)
    monkeypatch.setattr("studio_app.projects.background_recovery.recover_startup", held)
    return release


def test_project_request_waits_for_background_restore(tmp_path, monkeypatch):
    saved = ProjectService(tmp_path)
    saved.add_template("Waited")
    release = hold_recovery(monkeypatch)
    app = make_app(tmp_path)
    try:
        client = signed_in(app)
        assert client.get("/api/v1/preferences").status_code == 200  # Never waits.
        assert not app.extensions["startup"].done.is_set()
        threading.Timer(0.3, release.set).start()
        project = client.get("/api/v1/project").get_json()
        assert [ied["name"] for ied in project["ieds"]] == ["Waited"]
        assert project["startup_recovery"]["status"] == "restored"
    finally:
        release.set()
        app.extensions["runtime"].close()


def test_recovery_that_stopped_to_protect_autosave_refuses_work(tmp_path, monkeypatch):
    (tmp_path / "autosave.subsim").write_bytes(b"damaged")

    def fail(*_args):
        raise PermissionError("injected backup failure")
    monkeypatch.setattr("studio_app.projects.startup_recovery.shutil.copyfile", fail)
    app = make_app(tmp_path)
    try:
        client = signed_in(app)
        response = client.post("/api/v1/project/save")
        assert response.status_code == 503
        assert response.get_json()["code"] == "startup_recovery_failed"
        assert (tmp_path / "autosave.subsim").read_bytes() == b"damaged"
    finally:
        app.extensions["runtime"].close()


def test_license_check_answers_before_the_wait(tmp_path, monkeypatch):
    release = hold_recovery(monkeypatch)
    app = make_app(tmp_path, license_service=UnlicensedService())
    try:
        client = signed_in(app)
        assert client.get("/api/v1/project").status_code == 402
        assert not app.extensions["startup"].done.is_set()
    finally:
        release.set()
        app.extensions["runtime"].close()
