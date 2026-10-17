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
    certificate.write_text(bundle(hwid=hwid.machine_id()), encoding="utf-8")
    error = start_error(ENFORCED / "studio_ied.exe")
    assert "license" not in error.lower() and "License" not in error


def test_worker_accepts_pre_v2_license_only_before_cutoff(certificate):
    certificate.write_text(bundle(hwid=hwid.legacy_machine_id()), encoding="utf-8")
    assert "license" not in start_error(ENFORCED / "studio_ied.exe").lower()
    late = bundle(hwid=hwid.legacy_machine_id(), issued_at=hwid.LEGACY_ISSUED_BEFORE)
    certificate.write_text(late, encoding="utf-8")
    error = start_error(ENFORCED / "studio_ied.exe")
    assert "another computer" in error or "rollback" in error


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
