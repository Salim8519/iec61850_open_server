"""The one-click development launcher reuses a verified live session."""
import importlib.util
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
SPEC = importlib.util.spec_from_file_location("studio_run_dev", ROOT / "studio/tools/run_dev.py")
run_dev = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(run_dev)


def test_existing_frontend_returns_only_verified_port(monkeypatch):
    monkeypatch.setattr(run_dev, "accepts_dev_session",
                        lambda url, token: url.endswith(":5175") and token == "dev")
    assert run_dev.existing_frontend(5173, 5199, "dev") == "http://127.0.0.1:5175"


def test_main_reopens_verified_session_instead_of_failing(monkeypatch, capsys):
    monkeypatch.setattr(run_dev, "require", lambda path, message: Path("available"))
    monkeypatch.setattr(run_dev, "require_free_port",
                        lambda port, service: (_ for _ in ()).throw(SystemExit("busy")))
    monkeypatch.setattr(run_dev, "existing_frontend",
                        lambda first, last, token: "http://127.0.0.1:5174")
    monkeypatch.setattr(run_dev, "start", lambda *args, **kwargs: (_ for _ in ()).throw(
        AssertionError("A second process tree must not start")))
    assert run_dev.main([]) == 0
    assert "http://127.0.0.1:5174/#session=dev" in capsys.readouterr().out
    assert not hasattr(run_dev, "webbrowser")
