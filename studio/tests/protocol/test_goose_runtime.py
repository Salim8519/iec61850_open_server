"""Native GOOSE readiness and private-channel safety (spec: docs/specs/goose.md)."""
import os
import time
from pathlib import Path

import pytest
from studio_app.runtime.worker_process import WorkerProcess
from studio_app.scl.importer import import_scl

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
ADAPTER = os.environ.get("STUDIO_GOOSE_ADAPTER", "")
pytestmark = pytest.mark.skipif(not (BIN / "studio_ied.exe").exists(), reason="Native build required")


def publisher_model(tmp_path):
    model = import_scl((ROOT / "scd/open_substation.scd").read_bytes(), "open_substation.scd", ["IED1_XCBR"])[0]
    config = tmp_path / "publisher.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    return model, config


def test_worker_reports_goose_readiness(tmp_path):
    model, config = publisher_model(tmp_path)
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    try:
        started = worker.request("start", cfg=str(config), ip="127.50.0.232", port=18132,
                                 controls=model["controls"], edition=2, goose={"publish": False})
        assert "goose_aware" in started["capabilities"]
        assert started["goose"] == {"compiled": True, "requested": False, "active": False, "control_blocks": 1}
    finally:
        worker.close()


@pytest.mark.skipif(not ADAPTER, reason="Set STUDIO_GOOSE_ADAPTER to a lab adapter GUID")
def test_frames_survive_vendor_console_output(tmp_path):
    model, config = publisher_model(tmp_path)
    events = []
    worker = WorkerProcess(BIN / "studio_ied.exe", events.append)
    try:
        started = worker.request("start", cfg=str(config), ip="127.50.0.233", port=18133,
                                 controls=model["controls"], edition=2,
                                 goose={"publish": True, "adapter_guid": ADAPTER})
        assert started["goose"]["active"], started["goose"]
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline and not any(
                event.get("event") == "goose_publishing" and event.get("state") == "active" for event in events):
            time.sleep(0.1)
        assert any(event.get("state") == "active" for event in events if event.get("event") == "goose_publishing")
        # The vendor Ethernet layer prints to the console while opening the adapter.
        # The private channel must still carry well-formed replies afterwards.
        for _ in range(20):
            assert worker.request("read", ref="IED1_XCBRGenericIO/XCBR1.Pos.stVal") in {0, 1, 2, 3}
        assert not any(event.get("event") == "worker_failed" for event in events)
    finally:
        worker.close()
