"""GOOSE published by studio_ied is received by studio_live on a lab adapter."""
import os
import time
from pathlib import Path

import pytest
from studio_app.runtime.worker_process import WorkerProcess
from studio_app.scl.importer import import_scl

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
ADAPTER = os.environ.get("STUDIO_GOOSE_ADAPTER", "")
pytestmark = [pytest.mark.skipif(not (BIN / "studio_live.exe").exists(), reason="Native build required"),
              pytest.mark.skipif(not ADAPTER, reason="Set STUDIO_GOOSE_ADAPTER to a lab adapter GUID")]


def wait_for(events, predicate, seconds):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        found = [event for event in list(events) if predicate(event)]
        if found:
            return found
        time.sleep(0.05)
    return []


def test_simulated_publisher_reaches_listener(tmp_path):
    model = import_scl((ROOT / "scd/open_substation.scd").read_bytes(), "open_substation.scd", ["IED1_XCBR"])[0]
    control = model["goose"]["publishes"][0]
    config = tmp_path / "publisher.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    received = []
    listener = WorkerProcess(BIN / "studio_live.exe", received.append)
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    try:
        listener.request("goose_start", timeout=30, adapter_guid=ADAPTER, subscriptions=[{
            "key": control["gocb_ref"], "gocb_ref": control["gocb_ref"], "app_id": control["app_id"],
            "mac": control["mac"], "go_id": control["go_id"], "dataset_ref": control["dataset_ref"],
            "conf_rev": control["conf_rev"]}])
        worker.request("start", cfg=str(config), ip="127.50.0.234", port=18134, controls=model["controls"],
                       edition=2, goose={"publish": True, "adapter_guid": ADAPTER})
        first = wait_for(received, lambda e: e.get("event") == "goose_rx" and e.get("state") == "receiving", 40)
        assert first, "No GOOSE frame received from the simulated publisher"
        frame = first[0]
        assert frame["conf_rev"] == control["conf_rev"] == 2
        assert frame["go_id"] == control["go_id"] == "events"
        assert frame["dataset"] == control["dataset_ref"]
        assert frame["problem"] == ""
        before = frame["st_num"]
        worker.request("update", ref="IED1_XCBRGenericIO/XCBR1.Pos.stVal", value=2)
        changed = wait_for(received, lambda e: e.get("event") == "goose_rx" and (e.get("st_num") or 0) > before, 10)
        assert changed, "stNum did not increase after the dataset value changed"
        assert changed[0]["values"] == [{"bits": 2, "le": 1, "be": 2}]
    finally:
        worker.close()
        listener.close()
