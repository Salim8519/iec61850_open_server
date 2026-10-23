"""Buffered and unbuffered report evidence from an independent TCP/MMS client."""
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
pytestmark = pytest.mark.skipif(not (BIN / "studio_probe.exe").exists(), reason="Native build required")


@pytest.mark.parametrize("rcb", ["RP.EventsRCB", "BR.Buffered"])
def test_gi_and_nested_float_data_change(tmp_path, rcb):
    model = import_scl(make_template("ReportTest"))[0]
    config = tmp_path / "report.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    client = None
    try:
        worker.request("start", cfg=str(config), ip="127.50.0.221", port=18104,
                       controls=model["controls"], edition=2)
        client = subprocess.Popen(
            [str(BIN / "studio_probe.exe"), "127.50.0.221", "18104", "reports",
             "ReportTestCTRL/LLN0." + rcb],
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        assert json.loads(client.stdout.readline())["ready"]
        worker.request("update", ref="ReportTestCTRL/GGIO1.AnIn1.mag.f", value=123.5)
        output, errors = client.communicate(timeout=10)
        assert client.returncode == 0, output + errors
        reports = json.loads(output)["reports"]
        entries = [entry for report in reports for entry in report["entries"]]
        assert any(entry["reason"] & 16 for entry in entries), "Missing GI report"
        assert any(entry["index"] == 3 and entry["reason"] & 1
                   and float(entry["value"]) == 123.5 for entry in entries), "Missing analog dchg"
    finally:
        if client:
            if client.poll() is None:
                client.kill()
            client.communicate()
        worker.close()


def test_report_created_only_for_an_enabled_block(tmp_path):
    """ADR-0115: a buffered RCB no client enabled raises no report_control events per change."""
    model = import_scl(make_template("ReportTest"))[0]
    config = tmp_path / "report.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    events = []
    worker = WorkerProcess(BIN / "studio_ied.exe", events.append)
    client = None
    created = lambda: [e for e in list(events) if e.get("event") == "report_control" and e.get("kind") == 9]
    try:
        worker.request("start", cfg=str(config), ip="127.50.0.223", port=18108,
                       controls=model["controls"], edition=2)
        for value in (1.5, 2.5, 3.5):
            worker.request("update", ref="ReportTestCTRL/GGIO1.AnIn1.mag.f", value=value)
        worker.request("read", ref="ReportTestCTRL/GGIO1.AnIn1.mag.f")  # events before this answer are in
        assert created() == [], "report created for a block nobody enabled"
        client = subprocess.Popen(
            [str(BIN / "studio_probe.exe"), "127.50.0.223", "18108", "reports", "ReportTestCTRL/LLN0.BR.Buffered"],
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        assert json.loads(client.stdout.readline())["ready"]
        worker.request("update", ref="ReportTestCTRL/GGIO1.AnIn1.mag.f", value=123.5)
        output, errors = client.communicate(timeout=10)
        assert client.returncode == 0, output + errors
        assert created(), "an enabled block still reports its created reports"
    finally:
        if client and client.poll() is None:
            client.kill()
        worker.close()
