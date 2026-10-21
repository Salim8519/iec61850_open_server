"""The native guard verifies the same certificates as Python and the same machine ID v3 as the Go engine."""
import json
import os
from pathlib import Path
import subprocess
import pytest
from license_signer import HWID, bundle, public_key
import hwid_v3
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


def test_native_machine_id_v3_matches_the_independent_reference():
    """ADR-0098: firmware UUID + board serial (MachineGuid only when both are placeholders)."""
    assert run("hwid").stdout.strip() == hwid_v3.machine_id()
    assert run("hwid").stdout.strip() != hwid.machine_id()  # nothing of the older v2 ID is left
    cases = json.loads((STUDIO / "tests/fixtures/licensing/hwid_v3.json").read_text(encoding="utf-8"))["cases"]
    assert len(cases) >= 10
    for case in cases:
        assert hwid_v3.compose(case["firmware"], case["board"], case["guid"]) == case["hwid"], case["name"]
        native = run("compose", case["firmware"], case["board"], case["guid"]).stdout.strip()
        assert native == case["hwid"], case["name"]


def test_native_reads_the_system_uuid_and_board_serial_like_the_reference():
    system = hwid_v3.structure(1, 27, "Maker", "Product", serial_index=2)
    system = system[:8] + bytes.fromhex("7a1b2c3d4e5f60718293a4b5c6d7e8f9") + system[24:]
    board = hwid_v3.structure(2, 15, "Board maker", "Board", "SN-4711", serial_index=3)
    cases = {
        "both": (hwid_v3.blob(hwid_v3.structure(0, 24, "Vendor"), system, board), "SN-4711"),
        "serial number 0 means none": (hwid_v3.blob(system, hwid_v3.structure(2, 15, "M", "B", serial_index=0)), ""),
        "serial number beyond the set": (hwid_v3.blob(hwid_v3.structure(2, 15, "M", serial_index=9)), ""),
        "baseboard without strings": (hwid_v3.blob(hwid_v3.structure(2, 15, serial_index=1)), ""),
        "first baseboard wins": (hwid_v3.blob(hwid_v3.structure(2, 15, "FIRST", serial_index=1),
                                              hwid_v3.structure(2, 15, "SECOND", serial_index=1)), "FIRST"),
        "no baseboard": (hwid_v3.blob(system), ""),
        "baseboard too short for a serial": (hwid_v3.blob(hwid_v3.structure(2, 6, "M")), ""),
        "truncated table": (bytes(8) + bytes([2, 15, 0, 0, 1]), ""),
    }
    for name, (raw, serial) in cases.items():
        expected_uuid = hwid_v3.system_uuid(raw)
        assert hwid_v3.board_serial(raw) == serial, name
        assert run("smbios", raw.hex()).stdout.strip() == expected_uuid + "|" + serial, name
    assert hwid_v3.system_uuid(cases["both"][0]) == "7a1b2c3d4e5f60718293a4b5c6d7e8f9"
    # The real table of this computer parses the same everywhere.
    real = hwid_v3.raw_table()
    if len(real) > 8:
        out = run("smbios", real.hex()).stdout.strip()
        assert out == hwid_v3.system_uuid(real) + "|" + hwid_v3.board_serial(real)


def test_native_verifies_signed_module_and_rejects_tampering(tmp_path):
    import hashlib
    from studio_app.licensing import ed25519_sign as es
    seed = bytes.fromhex("11" * 32)
    module = tmp_path / "studio_app.pyd"
    module.write_bytes(b"pretend compiled module bytes")
    signature = tmp_path / "studio_app.pyd.sig"
    signature.write_bytes(es.sign(hashlib.sha256(module.read_bytes()).hexdigest().encode(), seed))
    key = es.public_key(seed).hex()
    assert run("verify-module", module, signature, key).stdout.strip() == "ok"
    module.write_bytes(b"tampered compiled module!!!!!")
    assert run("verify-module", module, signature, key).returncode != 0


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
