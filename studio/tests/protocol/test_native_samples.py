"""Verify batched value/quality/time updates against the native model and MMS."""
import json
import os
import subprocess
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
