"""An independent MMS client (studio_probe) shows up in the worker's access summary (MMSACC-05..07)."""
import json
import os
import subprocess
import time
from pathlib import Path

import pytest
from studio_app.runtime.worker_process import WorkerProcess
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_probe.exe").exists(), reason="Native build required")
IP, PORT = "127.50.0.226", 18126


def probe(*args):
    completed = subprocess.run([str(BIN / "studio_probe.exe"), IP, str(PORT), *args],
                               capture_output=True, text=True, timeout=20)
    return json.loads(completed.stdout)


def wait_for(events, predicate, seconds):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        found = [event for event in list(events) if predicate(event)]
        if found:
            return found
        time.sleep(0.05)
    return []


def reads_of(events, node):
    return sum(client["reads"].get(node, 0) for event in events if event.get("event") == "mms_access"
               for client in event["clients"])


@pytest.fixture
def worker(tmp_path):
    model = import_scl(make_template("AccessTest"))[0]
    config = tmp_path / "access.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    events = []
    process = WorkerProcess(BIN / "studio_ied.exe", events.append)
    started = process.request("start", cfg=str(config), ip=IP, port=PORT, controls=model["controls"], edition=2)
    assert "mms_access" in started["capabilities"]
    yield events
    process.close()


def test_reads_are_counted_per_client_and_answers_are_unchanged(worker):
    events = worker
    for _ in range(3):
        result = probe("read", "AccessTestCTRL/GGIO1.AnIn1", "MX")
        assert result["ok"], result
    # Without the observer libiec61850 answers data access error 10 (object non-existent) for all three;
    # IedServer_setReadAccessHandler would turn the first two into "access unsupported" (ADR-0066).
    for missing in ("AccessTestCTRL/GGIO1.NoSuchObject", "AccessTestCTRL/NOLN1.Mod", "NoLD/LLN0.Mod"):
        assert probe("read", missing, "ST")["value"] == "error 10"
    assert wait_for(events, lambda event: reads_of([event], "AccessTestCTRL/GGIO1") > 0, 5)
    time.sleep(1.5)
    assert reads_of(events, "AccessTestCTRL/GGIO1") >= 3
    connections = [event for event in events if event.get("event") == "connection"]
    assert connections and all(event["peer"].startswith("127.") for event in connections)
    summaries = [event for event in events if event.get("event") == "mms_access"]
    assert len(summaries) <= 8, "the worker must summarise, not report every read"
    left = summaries[-1]["clients"]
    assert all(client["connected"] is False for client in left) or not left


def test_report_client_and_control_carry_the_peer(worker):
    events = worker
    client = subprocess.Popen([str(BIN / "studio_probe.exe"), IP, str(PORT), "reports",
                               "AccessTestCTRL/LLN0.RP.EventsRCB"], stdout=subprocess.PIPE,
                              stderr=subprocess.PIPE, text=True)
    try:
        assert json.loads(client.stdout.readline())["ready"]
        enabled = wait_for(events, lambda event: event.get("event") == "mms_access" and any(
            "LLN0.EventsRCB01" in item["rcbs"] or any("EventsRCB" in name for name in item["rcbs"])
            for item in event["clients"]), 5)
        assert enabled, [event for event in events if event.get("event") in {"mms_access", "report_control"}]
        assert probe("control", "AccessTestCTRL/GGIO1.SPCSO1", "true")["ok"]
        applied = wait_for(events, lambda event: event.get("event") == "control_applied", 5)
        assert applied and applied[0]["peer"].startswith("127.")
        written = wait_for(events, lambda event: event.get("event") == "mms_access" and any(
            item["writes"].get("AccessTestCTRL/GGIO1.SPCSO1") for item in event["clients"]), 5)
        assert written
    finally:
        client.kill()
        client.communicate()
