"""Station health: GOOSE with an unexpected goID is flagged on the wire (HEALTH-06)."""
import os
import time
from pathlib import Path

import pytest
from studio_app.domain.goose_consistency import wire_check
from studio_app.runtime.worker_process import WorkerProcess
from studio_app.scl.importer import import_scl

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
ADAPTER = os.environ.get("STUDIO_GOOSE_ADAPTER", "")
pytestmark = [pytest.mark.skipif(not (BIN / "studio_live.exe").exists(), reason="Native build required"),
              pytest.mark.skipif(not ADAPTER, reason="Set STUDIO_GOOSE_ADAPTER to a lab adapter GUID")]


def first(events, predicate, seconds):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        found = next((event for event in list(events) if predicate(event)), None)
        if found:
            return found
        time.sleep(0.05)
    return None


def test_go_id_mismatch_is_reported(tmp_path):
    model = import_scl((ROOT / "scd/open_substation.scd").read_bytes(), "open_substation.scd", ["IED1_XCBR"])[0]
    control = model["goose"]["publishes"][0]
    config = tmp_path / "publisher.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    received = []
    listener = WorkerProcess(BIN / "studio_live.exe", received.append)
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    expected = {"key": control["gocb_ref"], "gocb_ref": control["gocb_ref"], "app_id": control["app_id"],
                "mac": control["mac"], "go_id": "not-" + control["go_id"], "dataset_ref": control["dataset_ref"],
                "conf_rev": control["conf_rev"]}
    try:
        listener.request("goose_start", timeout=30, adapter_guid=ADAPTER, subscriptions=[expected])
        worker.request("start", cfg=str(config), ip="127.50.0.235", port=18135, controls=model["controls"],
                       edition=2, goose={"publish": True, "adapter_guid": ADAPTER})
        frame = first(received, lambda e: e.get("event") == "goose_rx" and e.get("state") == "problem", 40)
        assert frame, "The listener never reported the goID difference"
        assert frame["problem"] == "go_id_mismatch" and frame["go_id"] == control["go_id"]
        watch = {**expected, "state": frame["state"], "problem": frame["problem"], "received_go_id": frame["go_id"],
                 "received_conf_rev": frame["conf_rev"], "received_dataset": frame["dataset"], "observed_at": "now"}
        verdict = wire_check(watch, {"state": "listening"})
        assert verdict["result"] == "problem"
        assert f"wire {control['go_id']}" in verdict["message"] and expected["go_id"] in verdict["message"]
    finally:
        worker.close()
        listener.close()
