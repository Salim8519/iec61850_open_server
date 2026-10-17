"""Release guards: the installed app refuses developer mode and reports its version."""
import sys
import pytest
from studio_app.app_factory import create_app
from studio_app.launcher import entrypoint
from studio_app.licensing.service import LicenseService
from studio_app.settings import Settings
from studio_app.version import __version__
from license_signer import HWID, keys


@pytest.fixture
def launcher(monkeypatch, tmp_path):
    """Stop main() before it serves: record the dev reloader call instead of running it."""
    reloads = []
    monkeypatch.setattr(entrypoint, "attach_parent_terminal", lambda: None)
    monkeypatch.setattr(entrypoint, "configure_logging", lambda directory: None)
    monkeypatch.setattr("werkzeug._reloader.run_with_reloader", lambda *args, **kwargs: reloads.append(args))
    monkeypatch.setattr(sys, "argv", ["ScadaBench", "--dev", "--data-dir", str(tmp_path)])
    return reloads


def test_installed_app_refuses_developer_mode(launcher, monkeypatch):
    monkeypatch.setattr(sys, "frozen", True, raising=False)
    with pytest.raises(SystemExit, match="Developer mode"):
        entrypoint.main()
    assert launcher == []


def test_developer_mode_still_runs_from_source(launcher, monkeypatch):
    monkeypatch.delattr(sys, "frozen", raising=False)
    entrypoint.main()
    assert len(launcher) == 1


@pytest.fixture
def client(tmp_path):
    """A signed-in packaged backend that has no license yet."""
    settings = Settings(tmp_path / "data", tmp_path / "unused.exe", tmp_path)
    service = LicenseService(tmp_path / "home", settings.directory, True, hwid_provider=lambda: HWID,
                             clock=lambda: 1_760_000_100, keys=keys())
    app = create_app(settings, "version-test", license_service=service)
    client = app.test_client()
    assert client.post("/api/v1/session", json={"token": "version-test"}).status_code == 200
    yield client
    app.extensions["runtime"].close()


def test_unlicensed_status_still_reports_the_app_version(client):
    assert client.get("/api/v1/project").status_code == 402
    response = client.get("/api/v1/license/status")
    assert response.status_code == 200 and response.json["app_version"] == __version__
    assert not response.json["licensed"]


def test_health_reports_the_app_version(tmp_path):
    app = create_app(Settings(tmp_path, tmp_path / "unused.exe", tmp_path), "health-test")
    try:
        client = app.test_client()
        assert client.post("/api/v1/session", json={"token": "health-test"}).status_code == 200
        response = client.get("/api/v1/health")
        assert response.status_code == 200 and response.json["app_version"] == __version__
    finally:
        app.extensions["runtime"].close()
