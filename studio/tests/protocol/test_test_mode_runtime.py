"""Test mode writes reach a simulated IED, read by an independent client (TMODE-05, TMODE-10)."""
import json
import os
import subprocess
from pathlib import Path

import pytest
from studio_app.domain.test_mode import mode_writes, validate_test_mode
from studio_app.runtime.worker_process import WorkerProcess
from studio_app.scl.importer import import_scl

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_ied.exe").exists(), reason="Native build required")
HOST, PORT = "127.50.0.242", 18142


def probe(ref):
    out = subprocess.run([str(BIN / "studio_probe.exe"), HOST, str(PORT), "read", ref, "ST"],
                         capture_output=True, text=True, timeout=30)
    return json.loads(out.stdout)["value"]


@pytest.fixture
def device(tmp_path):
    model = import_scl((ROOT / "scd/open_substation.scd").read_bytes(), "open_substation.scd", ["IED1_XCBR"])[0]
    config = tmp_path / "device.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    worker.request("start", cfg=str(config), ip=HOST, port=PORT, controls=model["controls"], edition=2)
    yield model, worker
    worker.close()


def test_integer_and_enum_attributes_are_written(device):
    _, worker = device
    ref = "IED1_XCBRGenericIO/LLN0.Beh.stVal"
    assert worker.request("update_batch", updates=[{"ref": ref, "value": 4}]) == {ref: 4}
    assert probe(ref) == "4"
    with pytest.raises(RuntimeError, match="out of range"):
        worker.request("update_batch", updates=[{"ref": ref, "value": 2 ** 40}])
    assert probe(ref) == "4"


def test_mode_writes_reach_the_simulated_ied(device):
    model, worker = device
    writes = mode_writes(model, validate_test_mode({"devices": {"IED1_XCBRGenericIO": 3}}), {})
    worker.request("update_batch", updates=[{"ref": ref, "value": value} for ref, value in writes])
    assert probe("IED1_XCBRGenericIO/LLN0.Beh.stVal") == "3"
    assert probe("IED1_XCBRGenericIO/XCBR1.Beh.stVal") == "3"
