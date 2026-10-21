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


@pytest.fixture
def packaged_module(monkeypatch, tmp_path):
    """A packaged start whose compiled module lives in tmp_path; returns (module, messages)."""
    import studio_app
    module = tmp_path / "studio_app.cp312-win_amd64.pyd"
    module.write_bytes(b"compiled module")
    messages = []
    monkeypatch.setattr(studio_app, "__file__", str(module))
    monkeypatch.setattr(sys, "frozen", True, raising=False)
    monkeypatch.setenv("LOCALAPPDATA", str(tmp_path / "local"))
    monkeypatch.delenv("STUDIO_MODULE_PATH", raising=False)
    monkeypatch.delenv("STUDIO_MODULE_SIG", raising=False)
    monkeypatch.setattr(entrypoint, "attach_parent_terminal", lambda: None)
    monkeypatch.setattr(entrypoint, "tell_user", lambda text, error=False: messages.append((text, error)))
    monkeypatch.setattr(sys, "argv", ["ScadaBench", "--self-test", "--data-dir", str(tmp_path / "data")])
    return module, messages


@pytest.mark.parametrize("signature", [None, b""])
def test_installed_app_refuses_to_start_without_a_readable_signature(packaged_module, signature, monkeypatch):
    module, messages = packaged_module
    if signature is not None:
        (module.parent / (module.name + ".sig")).write_bytes(signature)
    with pytest.raises(SystemExit) as stopped:
        entrypoint.main()
    assert stopped.value.code == 1 and messages == [(entrypoint.SIGNATURE_MISSING, True)]
    assert "STUDIO_MODULE_PATH" not in __import__("os").environ


def test_installed_app_refuses_an_uncompiled_module_even_with_a_signature(packaged_module):
    module, messages = packaged_module
    source = module.parent / "studio_app" / "__init__.py"
    source.parent.mkdir()
    source.write_text("", encoding="utf-8")
    (source.parent / "__init__.py.sig").write_bytes(b"signature")
    import studio_app
    studio_app.__file__ = str(source)
    assert entrypoint._announce_signed_module() == entrypoint.SIGNATURE_MISSING


def test_a_signed_module_is_announced_to_the_native_workers(packaged_module):
    import os
    module, messages = packaged_module
    (module.parent / (module.name + ".sig")).write_bytes(b"signature")
    assert entrypoint._announce_signed_module() is None
    assert os.environ["STUDIO_MODULE_PATH"] == str(module) and os.environ["STUDIO_MODULE_SIG"] == str(module) + ".sig"
    for name in ("STUDIO_MODULE_PATH", "STUDIO_MODULE_SIG"):
        os.environ.pop(name)
