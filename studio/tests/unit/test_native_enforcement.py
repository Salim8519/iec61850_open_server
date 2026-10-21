"""Release-configured workers refuse MMS/GOOSE/SV until this machine has a valid license.

Build the enforced binaries with the test key (see docs/changes/2026/0069-licensing.md):
    configure_release.py ... --native-out studio/native/build/license-keys
    cmake ... -DSTUDIO_ENFORCE_LICENSE=ON -DSTUDIO_RELEASE_KEY_DIR=.../license-keys
"""
import os
from pathlib import Path
import subprocess
import pytest
from license_signer import bundle
import hwid_v3
from studio_app.licensing import hwid
from studio_app.runtime.worker_process import WorkerProcess

STUDIO = Path(__file__).resolve().parents[2]
ENFORCED = Path(os.environ.get("STUDIO_ENFORCED_NATIVE_BIN", STUDIO / "native/build/license-enforced/Release"))
pytestmark = pytest.mark.skipif(not (ENFORCED / "studio_ied.exe").exists(), reason="enforced native build not present")


@pytest.fixture
def certificate(tmp_path, monkeypatch):
    path = tmp_path / "license.cert"
    monkeypatch.setenv("STUDIO_LICENSE_CERT_PATH", str(path))
    return path


def start_error(executable):
    worker = WorkerProcess(executable)
    try:
        worker.request("start", timeout=10, ieds=[])
    except RuntimeError as error:
        return str(error)
    finally:
        worker.close()
    return ""


def test_enforced_build_reports_enforcement():
    output = subprocess.run([str(ENFORCED / "studio_license_check.exe"), "enforced"], capture_output=True, text=True)
    assert output.stdout.strip() == "1"


def test_worker_refuses_to_serve_without_license(certificate):
    assert "No license" in start_error(ENFORCED / "studio_ied.exe")
    certificate.write_text(bundle(hwid="cd" * 32), encoding="utf-8")
    assert "another computer" in start_error(ENFORCED / "studio_ied.exe")


def test_worker_passes_license_gate_with_valid_certificate(certificate):
    certificate.write_text(bundle(hwid=hwid_v3.machine_id()), encoding="utf-8")
    error = start_error(ENFORCED / "studio_ied.exe")
    assert "license" not in error.lower() and "License" not in error


def test_worker_refuses_a_cancelled_license_whatever_the_certificate_says(certificate):
    certificate.write_text(bundle(hwid=hwid_v3.machine_id()), encoding="utf-8")
    assert "license" not in start_error(ENFORCED / "studio_ied.exe").lower()
    revoked = certificate.parent / ".revoked_ids"
    revoked.write_text("SUBS-OTHR-OTHR-OTHR-OTHR,SUBS-TEST-TEST-TEST-TEST", encoding="utf-8")  # ADR-0097
    error = start_error(ENFORCED / "studio_ied.exe")
    assert "License check failed" in error and "cancelled" in error
    revoked.unlink()
    state = certificate.parent / "license_state.json"
    state.write_text('{"revoked_ids": "SUBS-TEST-TEST-TEST-TEST"}', encoding="utf-8")  # one copy is enough
    assert "cancelled" in start_error(ENFORCED / "studio_ied.exe")


def test_worker_accepts_only_the_machine_id_v3(certificate):
    """ADR-0098: certificates for the older v1 and v2 machine IDs are refused, whenever they were issued."""
    for old in (hwid.machine_id(), hwid.legacy_machine_id()):
        certificate.write_text(bundle(hwid=old), encoding="utf-8")
        assert "another computer" in start_error(ENFORCED / "studio_ied.exe")
    certificate.write_text(bundle(hwid=hwid.legacy_machine_id(), issued_at=1_000_000_000), encoding="utf-8")
    assert "another computer" in start_error(ENFORCED / "studio_ied.exe")


def test_worker_applies_the_trial_online_rule(certificate):
    """ADR-0046 addendum 2026-10-19: no signed time yet = grace; grace over = online check."""
    import time
    now, day = int(time.time()), 86400
    certificate.write_text(bundle(type="trial", hwid=hwid_v3.machine_id(), issued_at=now - day,
                                  expires_at=now + 6 * day, license_id="TRIAL-test"), encoding="utf-8")
    assert "license" not in start_error(ENFORCED / "studio_ied.exe").lower()
    (certificate.parent / ".grace_start").write_text(str(now - 4 * day))
    assert "online check" in start_error(ENFORCED / "studio_ied.exe")


def test_live_client_and_probe_refuse_without_license(certificate):
    worker = WorkerProcess(ENFORCED / "studio_live.exe")
    try:
        with pytest.raises(RuntimeError, match="No license"):
            worker.request("connect", timeout=10, host="127.0.0.1", port=102)
        assert "capabilities" in worker.request("hello", timeout=10)
    finally:
        worker.close()
    probe = subprocess.run([str(ENFORCED / "studio_probe.exe"), "127.0.0.1", "102", "browse"],
                           capture_output=True, text=True, timeout=10)
    assert probe.returncode == 4 and "No license" in probe.stderr


def test_gateway_refuses_to_listen_without_license(certificate):
    import socket
    worker = WorkerProcess(ENFORCED / "studio_gateway.exe")
    try:
        with pytest.raises(RuntimeError, match="No license"):
            worker.request("start", timeout=10, protocol="modbus", settings={"bind": "127.0.0.1", "port": 15099},
                           points=[{"id": "p", "kind": "single", "address": 0}])
        with pytest.raises(OSError):
            socket.create_connection(("127.0.0.1", 15099), timeout=1).close()
    finally:
        worker.close()


def test_control_client_refuses_to_operate_without_license(certificate):
    worker = WorkerProcess(ENFORCED / "studio_control.exe")
    try:
        with pytest.raises(RuntimeError, match="No license"):
            worker.request("operate", timeout=10, ip="127.0.0.1", port=1, ref="LD0/XCBR1.Pos", value=True,
                           or_cat=3, or_ident="test", interlock_check=True, synchro_check=False, test=False)
    finally:
        worker.close()


def test_master_refuses_to_connect_without_license(certificate):
    master = ENFORCED / "studio_master.exe"
    if not master.exists():
        pytest.skip("enforced studio_master not built")
    worker = WorkerProcess(master)
    try:
        with pytest.raises(RuntimeError, match="No license"):
            worker.request("connect", timeout=10, protocol="iec104", settings={"host": "127.0.0.1", "port": 1})
    finally:
        worker.close()
