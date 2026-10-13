"""studio_live against a separate studio_ied stand-in device (spec: docs/specs/real-ieds.md)."""
import os
import subprocess
import time
from pathlib import Path

import pytest
from studio_app.domain.model_compare import compare
from studio_app.runtime.worker_process import WorkerProcess
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_live.exe").exists(), reason="Native build required")
RCB = "ReportTestCTRL/LLN0.RP.EventsRCB"


@pytest.fixture
def device(tmp_path):
    model = import_scl(make_template("ReportTest"))[0]
    config = tmp_path / "device.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    worker.request("start", cfg=str(config), ip="127.50.0.231", port=18131, controls=model["controls"], edition=2)
    yield model, worker
    worker.close()


@pytest.fixture
def live():
    events = []
    process = WorkerProcess(BIN / "studio_live.exe", events.append)
    process.request("connect", ip="127.50.0.231", port=18131)
    yield process, events
    process.close()


def test_discover_and_read_simulated_stand_in(device, live):
    model, worker = device
    process, _ = live
    discovery = process.request("discover", timeout=60)
    result = compare(model, discovery)
    assert result["summary"]["missing_on_device"] == 0
    assert result["summary"]["type_mismatch"] == 0
    worker.request("update", ref="ReportTestCTRL/GGIO1.AnIn1.mag.f", value=42.5)
    answer = process.request("read", refs=[{"ref": "ReportTestCTRL/GGIO1.AnIn1.mag.f", "fc": "MX"},
                                          {"ref": "ReportTestCTRL/XCBR1.Pos.stVal", "fc": "ST"},
                                          {"ref": "ReportTestCTRL/GGIO1.Nope.stVal", "fc": "ST"}])
    by_ref = {item["ref"]: item for item in answer["values"]}
    assert by_ref["ReportTestCTRL/GGIO1.AnIn1.mag.f"]["value"] == pytest.approx(42.5)
    assert set(by_ref["ReportTestCTRL/XCBR1.Pos.stVal"]["value"]) == {"bits", "le", "be"}
    assert by_ref["ReportTestCTRL/GGIO1.Nope.stVal"]["error"] == "access_error"
    assert "value" not in by_ref["ReportTestCTRL/GGIO1.Nope.stVal"]


def test_write_operations_do_not_exist(device, live):
    process, _ = live
    for operation in ("write", "operate", "control", "set_value", "select", "setting_group"):
        with pytest.raises(RuntimeError, match="Unknown operation"):
            process.request(operation, ref="ReportTestCTRL/GGIO1.SPCSO1", value=True)


def test_enabled_block_is_not_taken(device):
    holder = subprocess.Popen([str(BIN / "studio_probe.exe"), "127.50.0.231", "18131", "reports", RCB],
                              stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    try:
        assert '"ready"' in holder.stdout.readline()
        process = WorkerProcess(BIN / "studio_live.exe", lambda event: None)
        try:
            process.request("connect", ip="127.50.0.231", port=18131)
            status = process.request("rcb_status", rcbs=[RCB])["blocks"][0]
            assert status["readable"] and status["enabled"] and not status["free"]
            assert process.request("rcb_reserve", rcb=RCB)["reserved"] is False
        finally:
            process.close()
    finally:
        holder.communicate(timeout=10)


def test_blocks_released_on_disconnect(device, live):
    model, worker = device
    process, events = live
    status = process.request("rcb_status", rcbs=[RCB])["blocks"][0]
    assert status["free"]
    reserved = process.request("rcb_reserve", rcb=RCB)
    assert reserved["reserved"]
    assert not process.request("rcb_status", rcbs=[RCB])["blocks"][0]["free"]
    worker.request("update", ref="ReportTestCTRL/GGIO1.AnIn1.mag.f", value=7.25)
    deadline = time.monotonic() + 5
    while time.monotonic() < deadline and not any(e.get("event") == "live_report" for e in events):
        time.sleep(0.05)
    reports = [event for event in events if event.get("event") == "live_report"]
    assert reports, "No report received after reservation"
    assert any("ReportTestCTRL/GGIO1.AnIn1.mag.f" in event["values"] for event in reports)
    process.request("disconnect")
    checker = WorkerProcess(BIN / "studio_live.exe", lambda event: None)
    try:
        checker.request("connect", ip="127.50.0.231", port=18131)
        after = checker.request("rcb_status", rcbs=[RCB])["blocks"][0]
        assert after["free"], after
    finally:
        checker.close()
