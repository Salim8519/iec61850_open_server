"""Owner license tool: local remove/activate never deletes files and targets the right folder."""
import importlib.util
import json
from pathlib import Path
from types import SimpleNamespace
from license_signer import bundle

STUDIO = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location("studio_license_admin", STUDIO / "tools/license_admin.py")
admin = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(admin)


def options(**values):
    defaults = dict(ssh=None, real=False, sandbox=True, email="owner@example.com", trial_days=0, key=None,
                    free_key=False, reset_trial=False)
    return SimpleNamespace(**{**defaults, **values})


def test_remove_moves_license_aside_and_frees_key(tmp_path, monkeypatch):
    monkeypatch.setattr(admin, "home", lambda args: tmp_path)
    (tmp_path / "license.cert").write_text(bundle(license_id="SUBS-AAAA-BBBB-CCCC-DDDD"))
    (tmp_path / ".time_anchor").write_text("1")
    calls = []
    monkeypatch.setattr(admin, "server", lambda args, *command: calls.append(command) or {"had_trial": True})
    admin.remove(options(free_key=True, reset_trial=True))
    assert not (tmp_path / "license.cert").exists()
    kept = next(tmp_path.glob("removed-*"))
    assert (kept / "license.cert").exists() and (kept / ".time_anchor").exists()
    assert calls[0] == ("reset-hwid", "--key", "SUBS-AAAA-BBBB-CCCC-DDDD")
    assert calls[1][0] == "reset-trial"


def test_activate_here_writes_certificate_for_this_pc(tmp_path, monkeypatch):
    monkeypatch.setattr(admin, "home", lambda args: tmp_path / "home")
    certificate = json.loads(bundle())
    monkeypatch.setattr(admin, "server", lambda args, *command: {"certificate": certificate,
                                                                  "license": {"key": "SUBS-X"}})
    admin.activate_here(options())
    assert json.loads((tmp_path / "home/license.cert").read_text()) == certificate


def test_real_server_is_reached_through_ssh_admin_wrapper():
    command = admin.server_command(options(ssh="me@203.0.113.10"), "list")
    assert command[-4:] == ["me@203.0.113.10", "sudo", "/usr/local/bin/license-server-admin", "list"]
