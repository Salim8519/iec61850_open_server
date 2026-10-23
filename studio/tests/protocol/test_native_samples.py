"""Verify batched value/quality/time updates against the native model and MMS."""
import json
import os
import subprocess
from datetime import datetime, timezone
from pathlib import Path

import pytest
from studio_app.runtime.worker_process import WorkerProcess
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_ied.exe").exists(), reason="Native build required")


def test_sample_batch_and_rejection_before_mutation(tmp_path):
    model = import_scl(make_template("Sample"))[0]
    config = tmp_path / "sample.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    prefix = "SampleCTRL/GGIO1.AnIn1"
    values = {prefix + ".mag.f": 41.5, prefix + ".q": 1, prefix + ".t": 1780000000000}
    try:
        worker.request("start", cfg=str(config), ip="127.50.0.220", port=18105,
                       controls=model["controls"], edition=2)
        result = worker.request("update_batch", updates=[{"ref": ref, "value": value}
                                                          for ref, value in values.items()])
        assert result == values
        for ref, value in values.items():
            assert worker.request("read", ref=ref) == value
            observed = subprocess.run([str(BIN / "studio_probe.exe"), "127.50.0.220", "18105",
                                       "read", ref, "MX"], capture_output=True, text=True, timeout=10)
            assert observed.returncode == 0, observed.stdout + observed.stderr
            if ref.endswith(".f"):
                assert float(json.loads(observed.stdout)["value"]) == value
        with pytest.raises(RuntimeError):
            worker.request("update_batch", updates=[{"ref": prefix + ".mag.f", "value": 999},
                                                      {"ref": "Missing.Reference", "value": 1}])
        assert worker.request("read", ref=prefix + ".mag.f") == 41.5
    finally:
        worker.close()


def test_one_batch_holds_4096_attributes(tmp_path):
    """ADR-0115: a mode change on a large LD is one native batch (one data-model lock)."""
    model = import_scl(make_template("Sample"))[0]
    config = tmp_path / "sample.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    ref = "SampleCTRL/GGIO1.AnIn1.mag.f"
    try:
        worker.request("start", cfg=str(config), ip="127.50.0.221", port=18106,
                       controls=model["controls"], edition=2)
        updates = [{"ref": ref, "value": float(index)} for index in range(4096)]
        assert worker.request("update_batch", updates=updates) == {ref: 4095.0}
        with pytest.raises(RuntimeError, match="1 to 4096"):
            worker.request("update_batch", updates=updates + [{"ref": ref, "value": 1.0}])
        assert worker.request("read", ref=ref) == 4095.0
    finally:
        worker.close()


def test_timestamps_start_at_the_run_start(tmp_path):
    """ADR-0115: every "t" reads the start time at Run, not 1970; one TimeQuality (0xAA)."""
    model = import_scl(make_template("Sample"))[0]
    config = tmp_path / "sample.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    ref = "SampleCTRL/GGIO1.AnIn1.t"
    try:
        started = worker.request("start", cfg=str(config), ip="127.50.0.222", port=18107,
                                 controls=model["controls"], edition=2)
        assert started["time_quality"] == 0xAA and started["stamped_times"] > 0
        assert started["started_at"] > 1_700_000_000_000
        observed = subprocess.run([str(BIN / "studio_probe.exe"), "127.50.0.222", "18107", "read", ref, "MX"],
                                  capture_output=True, text=True, timeout=10)
        assert observed.returncode == 0, observed.stdout + observed.stderr
        at = datetime.fromtimestamp(started["started_at"] / 1000, timezone.utc)
        assert json.loads(observed.stdout)["value"] == at.strftime("%Y%m%d%H%M%S.") + f"{at.microsecond // 1000:03d}Z"
    finally:
        worker.close()
