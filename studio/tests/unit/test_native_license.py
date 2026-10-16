"""The native guard verifies the same certificates and machine ID as Python."""
import json
import os
from pathlib import Path
import subprocess
import pytest
from license_signer import HWID, bundle, public_key
from studio_app.licensing import hwid

STUDIO = Path(__file__).resolve().parents[2]
NATIVE_BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", STUDIO / "native/build/Release"))
CHECK = NATIVE_BIN / "studio_license_check.exe"
pytestmark = pytest.mark.skipif(not CHECK.exists(), reason="native license_check test tool not built")
NOW = 1_760_000_100


def run(*args):
    return subprocess.run([str(CHECK), *map(str, args)], capture_output=True, text=True, timeout=10)


def verify(tmp_path, text, machine=HWID, now=NOW, anchor=0, key=None):
    path = tmp_path / "license.cert"
    path.write_text(text, encoding="utf-8")
    return run("verify", path, (key or public_key()).hex(), machine, now, anchor)


def test_native_machine_id_matches_python():
    assert run("hwid").stdout.strip() == hwid.machine_id()
    assert run("compose", " 1A2B-CCCC ", "a1b2c3d4").stdout.strip() == hwid.compose(" 1A2B-CCCC ", 0xA1B2C3D4)


def test_native_accepts_go_and_python_signed_certificates(tmp_path):
    vector = json.loads((STUDIO / "tests/fixtures/licensing/go_vector.json").read_text())
    for name in ("lifetime", "trial"):
        result = verify(tmp_path, json.dumps(vector[name]), vector["hwid"], key=bytes.fromhex(vector["public_key"]))
        assert result.returncode == 0, result.stdout + result.stderr
    assert verify(tmp_path, bundle()).stdout.strip() == "ok"


def test_native_rejects_tamper_wrong_machine_expiry_and_rollback(tmp_path):
    tampered = json.loads(bundle())
    tampered["payload"] = tampered["payload"].replace("lifetime", "lifetimx")
    assert "changed" in verify(tmp_path, json.dumps(tampered)).stdout
    assert "another computer" in verify(tmp_path, bundle(), machine="cd" * 32).stdout
    trial = bundle(type="trial", expires_at=NOW + 10)
    assert "expired" in verify(tmp_path, trial, now=NOW + 10).stdout
    assert "rollback" in verify(tmp_path, bundle(), anchor=NOW + 3600).stdout
    assert verify(tmp_path, bundle(seed=bytes(32))).returncode == 1


def test_development_build_does_not_enforce():
    expected = os.environ.get("STUDIO_EXPECT_ENFORCED", "0")
    assert run("enforced").stdout.strip() == expected
