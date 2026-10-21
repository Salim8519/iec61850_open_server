"""The native guard refuses a cancelled license in every place Python does (ADR-0097).

Same files, two verdicts: Python's LicenseService and the native studio_license_check tool
read the dot-file `.revoked_ids` and the `revoked_ids` field of license_state.json (the
tool skips the registry), and a certificate whose ID is in ANY copy is refused.
"""
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
LICENSE_ID = "SUBS-TEST-TEST-TEST-TEST"
LATER = NOW + 30 * DAY


def native(tmp_path, now=LATER):
    home = tmp_path / "home"
    result = subprocess.run([str(CHECK), "verify-clock", home / "license.cert", public_key().hex(), HWID, str(now),
                             str(home)], capture_output=True, text=True, timeout=10)
    return result.stdout.strip()


def both(tmp_path, now=LATER):
    """(python licensed?, native verdict) for the same folder at the same time."""
    return studio(tmp_path, [now], Server(online=False)).is_licensed(), native(tmp_path, now)


def install(tmp_path, text=None):
    home = tmp_path / "home"
    home.mkdir(parents=True, exist_ok=True)
    (home / "license.cert").write_text(text or bundle(), encoding="utf-8")


def state_with(tmp_path, ids):
    state = tmp_path / "home/license_state.json"
    data = json.loads(state.read_text(encoding="utf-8")) if state.exists() else {}
    data["revoked_ids"] = ids
    state.write_text(json.dumps(data), encoding="utf-8")


def test_a_license_nobody_cancelled_is_fine_in_both(tmp_path):
    install(tmp_path)
    assert both(tmp_path) == (True, "ok")


def test_a_cancelled_id_in_the_dot_file_is_refused_by_both(tmp_path):
    install(tmp_path)
    (tmp_path / "home/.revoked_ids").write_text(LICENSE_ID, encoding="utf-8")
    licensed, verdict = both(tmp_path)
    assert not licensed and verdict.startswith("fail") and "cancelled" in verdict


def test_the_state_file_copy_alone_is_enough_in_the_native_guard(tmp_path):
    install(tmp_path)
    state_with(tmp_path, f"SUBS-OTHR-OTHR-OTHR-OTHR,{LICENSE_ID}")
    assert "cancelled" in native(tmp_path)  # Python restores the other copies, native just reads


def test_python_cancels_a_license_and_native_refuses_the_renamed_back_certificate(tmp_path):
    install(tmp_path)
    at = [LATER]
    assert studio(tmp_path, at, Server(time=LATER, verdict="revoked")).verify_online_if_due() == "revoked"
    assert not (tmp_path / "home/license.cert").exists()
    (aside,) = (tmp_path / "home").glob("license.cert.revoked-*")
    aside.rename(tmp_path / "home/license.cert")  # the cheat
    licensed, verdict = both(tmp_path)
    assert not licensed and "cancelled" in verdict
    (tmp_path / "home/.revoked_ids").unlink()  # one copy deleted: the other still refuses
    assert "cancelled" in native(tmp_path)


def test_messy_copies_are_read_the_same_way(tmp_path):
    install(tmp_path)
    (tmp_path / "home/.revoked_ids").write_text(f" ,SUBS-OTHR-OTHR-OTHR-OTHR, ,\n", encoding="utf-8")
    assert both(tmp_path) == (True, "ok")
    (tmp_path / "home/.revoked_ids").write_text(f"x, {LICENSE_ID} ,y\r\n", encoding="utf-8")
    licensed, verdict = both(tmp_path)
    assert not licensed and "cancelled" in verdict


def test_a_cancelled_trial_is_refused_by_both(tmp_path):
    install_trial(tmp_path)
    (tmp_path / "home/.revoked_ids").write_text("TRIAL-" + HWID[:12], encoding="utf-8")
    licensed, verdict = both(tmp_path, NOW + DAY)
    assert not licensed and "cancelled" in verdict


def test_an_altered_certificate_stays_a_signature_failure(tmp_path):
    original = json.loads(bundle())
    original["payload"] = original["payload"].replace('"lifetime"', '"lifetimf"')
    install(tmp_path, json.dumps(original))
    (tmp_path / "home/.revoked_ids").write_text(LICENSE_ID, encoding="utf-8")
    assert "cancelled" not in native(tmp_path) and native(tmp_path).startswith("fail")
