"""Smoke tests for the built folder; these are not clean-machine certification."""
import json
import os
import subprocess
import time
import urllib.error
import urllib.request
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[3]
FOLDER = Path(os.environ.get("STUDIO_PORTABLE_DIR", ROOT / "studio/dist/SubstationStudio"))
pytestmark = pytest.mark.skipif(not (FOLDER / "SubstationStudio.exe").exists(),
                                reason="Build portable folder first")


def test_frozen_server_serves_ui_and_protects_api(tmp_path):
    process = subprocess.Popen([str(FOLDER / "SubstationStudio.exe"), "--no-browser",
                                "--port", "0", "--data-dir", str(tmp_path)],
                               stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    try:
        metadata = tmp_path / "launcher.json"
        deadline = time.monotonic() + 20
        while not metadata.exists() and process.poll() is None and time.monotonic() < deadline:
            time.sleep(0.05)
        assert metadata.exists(), "Frozen launcher did not become ready"
        url = json.loads(metadata.read_text())["url"]
        with urllib.request.urlopen(url, timeout=5) as response:
            assert b"<div id=\"root\">" in response.read()
        with pytest.raises(urllib.error.HTTPError) as error:
            urllib.request.urlopen(url + "/api/v1/project", timeout=5)
        assert error.value.code == 401
        assert (FOLDER / "_internal/native/studio_ied.exe").is_file()
        assert (FOLDER / "_internal/native/studio_probe.exe").is_file()
        assert (FOLDER / "_internal/native/studio_live.exe").is_file()
        duplicate = subprocess.run([str(FOLDER / "SubstationStudio.exe"), "--no-browser",
                                    "--port", "0", "--data-dir", str(tmp_path)],
                                   capture_output=True, text=True, timeout=15)
        assert duplicate.returncode != 0
        assert "already open" in duplicate.stderr
    finally:
        # This test never starts workers. Terminate only its own preview process.
        process.terminate()
        process.communicate(timeout=10)


def test_frozen_schema_and_mms_self_test(tmp_path):
    result = subprocess.run([str(FOLDER / "SubstationStudio.exe"), "--self-test",
                             "--data-dir", str(tmp_path)], capture_output=True, text=True, timeout=40)
    assert result.returncode == 0, result.stdout + result.stderr
    output = json.loads(result.stdout)
    assert output["mms_value"] == 42.5
    assert output["array_value"] == 17.25
    assert output["edited_export_verified"]
    assert output["stopped_recovery_verified"]
    assert output["ied_deletion_verified"]
    assert output["saved_panel_and_pulse_verified"]
    assert not list(tmp_path.glob("studio-check-*"))
