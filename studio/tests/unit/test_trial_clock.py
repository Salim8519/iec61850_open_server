"""Trial clock (ADR-0046 addendum 2026-10-19), "smart, not soft": clear clock rollback is
caught, server outages and small drift never stop a trial, lifetime licenses untouched."""
import json
import sys
import pytest
from license_fakes import DAY, NOW, FakeRegistry, Server, install_trial, studio
from license_signer import bundle, keys, stamp
from studio_app.domain import license_policy as policy, trial_policy
from studio_app.licensing import server_time
from studio_app.licensing.clock_store import ClockValue, WindowsRegistry
from studio_app.licensing.client import LicenseServerError
from studio_app.licensing.service import LicenseActionError


def wipe_anchors(tmp_path, registry=None, stamps=False):
    """What a cheater deletes: the anchor copies (and optionally the signed time)."""
    home = tmp_path / "home"
    names = ["time_anchor", "server_time"] if stamps else ["time_anchor"]
    for name in names:
        (home / f".{name}").unlink(missing_ok=True)
        if registry:
            registry.values.pop({"time_anchor": "TimeAnchor", "server_time": "ServerTime"}[name], None)
    state = json.loads((home / "license_state.json").read_text())
    for name in names:
        state.pop(name, None)
    (home / "license_state.json").write_text(json.dumps(state))


def test_trial_start_keeps_the_signed_time_in_all_three_places(tmp_path):
    registry, at = FakeRegistry(), [NOW + 60]
    assert studio(tmp_path, at, Server(time=NOW + 60), registry).start_trial()["mode"] == "trial"
    home = tmp_path / "home"
    state = json.loads((home / "license_state.json").read_text())
    for copy in ((home / ".server_time").read_text(), registry.values["ServerTime"], state["server_time"]):
        assert server_time.verify(copy, keys()) == NOW + 60
    assert state["time_anchor"] == registry.values["TimeAnchor"] == (home / ".time_anchor").read_text()


def test_cheater_deleting_the_anchor_file_and_setting_the_clock_back_is_caught(tmp_path):
    registry, at = FakeRegistry(), [NOW + 5 * DAY]
    install_trial(tmp_path)
    assert studio(tmp_path, at, registry=registry).is_licensed()
    (tmp_path / "home/.time_anchor").unlink()  # the old loophole
    at[0] = NOW + 2 * DAY
    status = studio(tmp_path, at, registry=registry).status()
    assert not status["licensed"] and "rollback" in status["reason"] and policy.SUPPORT_URL in status["reason"]


def test_signed_time_catches_a_rollback_after_every_anchor_copy_is_deleted(tmp_path):
    registry, at = FakeRegistry(), [NOW + 5 * DAY]
    studio(tmp_path, at, Server(time=NOW + 5 * DAY), registry).start_trial()
    wipe_anchors(tmp_path, registry)
    at[0] = NOW + 2 * DAY
    assert "rollback" in studio(tmp_path, at, registry=registry).status()["reason"]


def test_small_clock_drift_never_stops_a_trial(tmp_path):
    """NTP corrections, a time-zone fix or a slow CMOS clock: up to 48 h back is fine."""
    at = [NOW + 4 * DAY]
    studio(tmp_path, at, Server(time=NOW + 4 * DAY)).start_trial()
    at[0] = NOW + 4 * DAY - 47 * 3600
    assert studio(tmp_path, at).is_licensed()
    at[0] = NOW + 4 * DAY - 49 * 3600
    assert "rollback" in studio(tmp_path, at).status()["reason"]


def test_lifetime_keeps_its_original_rollback_rule(tmp_path):
    home = tmp_path / "home"
    home.mkdir()
    (home / "license.cert").write_text(bundle(), encoding="utf-8")
    assert studio(tmp_path, [NOW + 3600]).is_licensed()
    assert not studio(tmp_path, [NOW + 3600 - 301]).is_licensed()  # unchanged 300 s, as before


def test_server_time_past_the_end_ends_the_trial_whatever_the_pc_clock_says(tmp_path):
    install_trial(tmp_path)
    at = [NOW + 7 * DAY - 3600]
    assert studio(tmp_path, at, Server(time=NOW + 7 * DAY + 3600)).verify_online_if_due() == "confirmed"
    assert studio(tmp_path, at).status()["reason"] == policy.TRIAL_ENDED


def test_server_down_never_stops_a_trial(tmp_path):
    at = [NOW]
    studio(tmp_path, at, Server(time=NOW)).start_trial()
    offline = Server(online=False)
    for day in (1, 3, 5, 6.9):
        at[0] = int(NOW + day * DAY)
        app = studio(tmp_path, at, offline)
        assert app.verify_online_if_due() is None and app.is_licensed()
    assert offline.calls.count("verify") >= 3  # tried quietly; failures never count


def test_trial_from_an_older_build_is_never_asked_anything(tmp_path):
    install_trial(tmp_path)
    (tmp_path / "home/.time_anchor").write_text(str(NOW + 600))  # what older builds kept; no signed time
    assert studio(tmp_path, [NOW + DAY]).is_licensed()
    status = studio(tmp_path, [NOW + 6 * DAY], Server(online=False)).status()
    assert status["licensed"] and not status["online_check_required"]


def test_missing_anchor_asks_one_online_check_within_three_days(tmp_path):
    at = [NOW + DAY]
    studio(tmp_path, at, Server(time=NOW + DAY)).start_trial()
    wipe_anchors(tmp_path, stamps=True)
    assert studio(tmp_path, at).is_licensed()  # suspicious, never an instant block
    at[0] = NOW + 4 * DAY - 60
    assert studio(tmp_path, at, Server(online=False)).is_licensed()
    at[0] = NOW + 4 * DAY + 60
    status = studio(tmp_path, at).status()
    assert status["online_check_required"] and "license.scadabench.com" in status["reason"]
    with pytest.raises(LicenseServerError) as offline:
        studio(tmp_path, at, Server(online=False)).check_online_now()
    assert offline.value.code == "offline"
    assert studio(tmp_path, at, Server(time=at[0])).check_online_now()["licensed"]


def test_missing_anchor_is_cleared_quietly_when_the_server_is_reachable(tmp_path):
    at = [NOW + DAY]
    studio(tmp_path, at, Server(time=NOW + DAY)).start_trial()
    wipe_anchors(tmp_path, stamps=True)
    assert studio(tmp_path, at, Server(time=NOW + DAY)).verify_online_if_due() == "confirmed"
    assert studio(tmp_path, [NOW + 6 * DAY], Server(online=False)).is_licensed()


def test_blocked_writes_everywhere_never_stop_a_trial(tmp_path, monkeypatch):
    """Antivirus / Controlled Folder Access: nothing can be saved, so nothing is held against the user."""
    install_trial(tmp_path)
    monkeypatch.setattr(ClockValue, "write", lambda self, text: None)
    for day in (1, 4, 6.9):
        assert studio(tmp_path, [int(NOW + day * DAY)], Server(online=False)).is_licensed()


def test_lifetime_license_never_needs_the_daily_check(tmp_path):
    home = tmp_path / "home"
    home.mkdir()
    (home / "license.cert").write_text(bundle(), encoding="utf-8")
    server, at = Server(online=False), [NOW + 3650 * DAY]
    app = studio(tmp_path, at, server)
    status = app.status()
    assert status["licensed"] and status["mode"] == "lifetime" and not status["online_check_required"]
    assert app.verify_online_if_due(trial_only=True) is None and server.calls == []
    assert not (home / ".grace_start").exists() and not (home / ".server_time").exists()


def test_stamps_and_certificates_can_never_stand_in_for_each_other(tmp_path):
    assert server_time.verify(json.loads(bundle()), keys()) is None
    assert server_time.verify(stamp(NOW, seed=bytes(32)), keys()) is None
    altered = stamp(NOW)
    altered["payload"] = altered["payload"].replace(str(NOW), str(NOW + 99 * DAY))
    assert server_time.verify(altered, keys()) is None
    with pytest.raises(LicenseActionError):
        studio(tmp_path, [NOW]).activate_offline(json.dumps(stamp(NOW)))


def test_email_step_passes_the_address_and_code_through(tmp_path):
    server = Server(needs_email=True)
    app = studio(tmp_path, [NOW], server)
    with pytest.raises(LicenseServerError) as asked:
        app.start_trial()
    assert asked.value.code == "email_required"
    assert app.request_trial_code("me@company.com") == {"status": "sent"}
    assert app.start_trial("me@company.com", "123456")["mode"] == "trial"
    with pytest.raises(LicenseActionError):
        app.request_trial_code("not an address")
    assert trial_policy.EMAIL_REQUIRED.count(policy.SUPPORT_URL) == 1


def test_blocked_places_are_ignored_silently(tmp_path):
    blocked = tmp_path / "not-a-folder"
    blocked.write_text("a file where the license folder should be")
    value = ClockValue("time_anchor", blocked, None)
    value.write(NOW)
    assert value.read_all() == []


@pytest.mark.skipif(sys.platform != "win32", reason="Windows registry only")
def test_windows_registry_place_round_trips_under_hkcu():
    import uuid
    import winreg
    subkey = rf"Software\ScadaBench-test-{uuid.uuid4().hex}"
    registry = WindowsRegistry(subkey)
    try:
        assert registry.get("TimeAnchor") is None
        registry.set("TimeAnchor", NOW)
        assert registry.get("TimeAnchor") == str(NOW)
    finally:
        winreg.DeleteKey(winreg.HKEY_CURRENT_USER, subkey)
