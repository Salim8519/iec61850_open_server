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
                    reset_trial=False, machine_id=None, ip=None)
    return SimpleNamespace(**{**defaults, **values})


def test_remove_moves_license_aside_and_never_frees_the_key(tmp_path, monkeypatch):
    monkeypatch.setattr(admin, "home", lambda args: tmp_path)
    (tmp_path / "license.cert").write_text(bundle(license_id="SUBS-AAAA-BBBB-CCCC-DDDD"))
    (tmp_path / ".time_anchor").write_text("1")
    (tmp_path / ".server_time").write_text("{}")
    calls = []
    monkeypatch.setattr(admin, "server", lambda args, *command: calls.append(command) or {"had_trial": True})
    (tmp_path / ".revoked_ids").write_text("SUBS-OLD-OLD-OLD-OLD")  # a cancelled license is never part of "remove"
    admin.remove(options(reset_trial=True))
    assert not (tmp_path / "license.cert").exists()
    kept = next(tmp_path.glob("removed-*"))
    assert (kept / "license.cert").exists() and (kept / ".time_anchor").exists() and (kept / ".server_time").exists()
    assert (tmp_path / ".revoked_ids").read_text() == "SUBS-OLD-OLD-OLD-OLD"
    assert not any(call[0] == "reset-hwid" for call in calls) and calls[0][0] == "reset-trial"


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


def test_owner_trial_overrides_send_this_pcs_parts_or_a_customer_machine_id(monkeypatch):
    calls = []
    monkeypatch.setattr(admin, "server", lambda args, *command: calls.append(command) or {"had_trial": True})
    monkeypatch.setattr(admin.hwid, "machine_id", lambda: "ab" * 32)
    monkeypatch.setattr(admin.trial_parts, "machine_parts", lambda: {"guid": "g" * 64, "volume": "", "firmware": "f" * 64})
    admin.reset_trial(options())
    assert calls[-1] == ("reset-trial", "--hwid", "ab" * 32, "--guid-hash", "g" * 64, "--firmware-hash", "f" * 64)
    admin.reset_trial(options(machine_id="cd" * 32))
    assert calls[-1] == ("reset-trial", "--hwid", "cd" * 32)
    admin.reset_trial_ip(options(ip="203.0.113.7"))
    assert calls[-1] == ("reset-trial-ip", "--ip", "203.0.113.7")
    admin.allow_email(options(email="Name@Company.com"))
    assert calls[-1] == ("allow-email", "--trial-email", "Name@Company.com")
