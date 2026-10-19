"""The native guard reaches the same trial-clock verdicts as Python on the same files."""
import json
import os
from pathlib import Path
import subprocess
import pytest
from license_fakes import DAY, NOW, Server, install_trial, studio
from license_signer import HWID, bundle, public_key

STUDIO = Path(__file__).resolve().parents[2]
CHECK = Path(os.environ.get("STUDIO_NATIVE_BIN", STUDIO / "native/build/Release")) / "studio_license_check.exe"
pytestmark = pytest.mark.skipif(not CHECK.exists(), reason="native license_check test tool not built")


def native(tmp_path, now):
    home = tmp_path / "home"
    result = subprocess.run([str(CHECK), "verify-clock", home / "license.cert", public_key().hex(), HWID, str(now),
                             str(home)], capture_output=True, text=True, timeout=10)
    return result.stdout.strip()


def both(tmp_path, now):
    """(python licensed?, native verdict) for the same folder at the same time."""
    return studio(tmp_path, [now]).is_licensed(), native(tmp_path, now)


def drop(tmp_path, *names):
    home = tmp_path / "home"
    state = json.loads((home / "license_state.json").read_text())
    for name in names:
        (home / f".{name}").unlink(missing_ok=True)
        state.pop(name, None)
    (home / "license_state.json").write_text(json.dumps(state))


def test_server_down_never_stops_a_trial_in_either(tmp_path):
    studio(tmp_path, [NOW], Server(time=NOW)).start_trial()
    assert both(tmp_path, NOW + 3 * DAY) == (True, "ok")
    assert both(tmp_path, int(NOW + 6.9 * DAY)) == (True, "ok")


def test_rollback_after_deleting_the_anchor_file(tmp_path):
    install_trial(tmp_path)
    (tmp_path / "home/.time_anchor").write_text(str(NOW + 5 * DAY))
    (tmp_path / "home/license_state.json").write_text(json.dumps({"time_anchor": str(NOW + 5 * DAY)}))
    (tmp_path / "home/.time_anchor").unlink()
    licensed, verdict = both(tmp_path, NOW + 2 * DAY)
    assert not licensed and "rollback" in verdict


def test_rollback_caught_by_signed_time_alone(tmp_path):
    studio(tmp_path, [NOW + 5 * DAY], Server(time=NOW + 5 * DAY)).start_trial()
    drop(tmp_path, "time_anchor")
    licensed, verdict = both(tmp_path, NOW + 2 * DAY)
    assert not licensed and "rollback" in verdict


def test_small_drift_is_allowed_and_big_rollback_is_not_in_either(tmp_path):
    studio(tmp_path, [NOW + 4 * DAY], Server(time=NOW + 4 * DAY)).start_trial()
    assert both(tmp_path, NOW + 4 * DAY - 47 * 3600) == (True, "ok")
    licensed, verdict = both(tmp_path, NOW + 4 * DAY - 49 * 3600)
    assert not licensed and "rollback" in verdict


def test_trial_ended_by_server_time(tmp_path):
    install_trial(tmp_path)
    studio(tmp_path, [NOW + 7 * DAY - 3600], Server(time=NOW + 7 * DAY + 60)).verify_online_if_due()
    licensed, verdict = both(tmp_path, NOW + 7 * DAY - 3600)
    assert not licensed and "expired" in verdict


def test_missing_anchor_asks_one_online_check_in_both(tmp_path):
    studio(tmp_path, [NOW + DAY], Server(time=NOW + DAY)).start_trial()
    drop(tmp_path, "time_anchor", "server_time")
    assert both(tmp_path, NOW + DAY) == (True, "ok")  # Python records when it noticed
    assert both(tmp_path, NOW + 4 * DAY - 60) == (True, "ok")
    licensed, verdict = both(tmp_path, NOW + 4 * DAY + 60)
    assert not licensed and "online check" in verdict


def test_forged_stamp_is_ignored_by_both(tmp_path):
    install_trial(tmp_path)
    home = tmp_path / "home"
    (home / ".time_anchor").write_text(str(NOW))
    forged = {"payload": json.dumps({"kid": "x", "purpose": "server-time", "time": NOW + 6 * DAY, "version": 1}),
              "signature": "AAAA"}
    (home / ".server_time").write_text(json.dumps(forged))
    assert both(tmp_path, NOW + DAY) == (True, "ok")  # accepted, it would read as a 5-day rollback


def test_lifetime_offline_for_years_is_fine_in_both(tmp_path):
    home = tmp_path / "home"
    home.mkdir()
    (home / "license.cert").write_text(bundle(), encoding="utf-8")
    assert both(tmp_path, NOW + 3650 * DAY) == (True, "ok")
