"""One GOOSE send = one batch = one stNum (ADR-0112, change 0183).

The Go engine writes every value of one send (a switch's open and closed contacts, or one value)
in ONE update_batch to the simulated IED. libiec61850 publishes pending GOOSE when the data model
lock of the batch is released, so the receiver must see the state number rise exactly once, with
both new values in that one frame. studio_live decodes the frames independently of the sender.
"""
import os
import time
from pathlib import Path

import pytest
from studio_app.runtime.worker_process import WorkerProcess
from studio_app.scl.importer import import_scl

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_goose.exe").exists() or not (BIN / "studio_live.exe").exists(),
                                reason="Native build and GOOSE engine required")
FEED = "FEED1LD1"


def frames(events):
    return [event for event in list(events) if event.get("event") == "goose_rx" and event.get("state") == "receiving"]


def wait_for(events, predicate, seconds):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        found = [event for event in frames(events) if predicate(event)]
        if found:
            return found
        time.sleep(0.05)
    return []


def test_two_values_in_one_batch_raise_st_num_once(tmp_path):
    model = import_scl((ROOT / "scd/model_substation.scd").read_bytes(), "model_substation.scd", ["FEED1"])[0]
    config = tmp_path / "feed1.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    control = model["goose"]["publishes"][0]
    received, engine_events = [], []
    engine = WorkerProcess(BIN / "studio_goose.exe", engine_events.append)
    listener = WorkerProcess(BIN / "studio_live.exe", received.append)
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    try:
        opened = engine.request("open", ieds=[{"id": "FEED1", "gocb_refs": [control["gocb_ref"]]}])
        mac = opened["sources"]["FEED1"]
        inbox = listener.request("goose_start", timeout=30, subscriptions=[{
            "key": control["gocb_ref"], "gocb_ref": control["gocb_ref"], "app_id": control["app_id"], "mac": control["mac"],
            "go_id": control["go_id"], "dataset_ref": control["dataset_ref"], "conf_rev": control["conf_rev"]}])["inbox_port"]
        engine.request("deliver", port=inbox)
        started = worker.request("start", cfg=str(config), ip="127.50.0.231", port=18131, controls=model["controls"],
                                 edition=2, goose={"publish": True, "interface": f"udp:{opened['port']}:{mac.replace('-', '')}"})
        assert started["goose"]["active"] is True
        first = wait_for(received, lambda event: True, 20)
        assert first, "No GOOSE frame reached studio_live through the engine"
        before = first[-1]["st_num"]
        # What Go sends for a disconnector "contacts" link or two values of one message: one batch.
        worker.request("update_batch", updates=[{"ref": f"{FEED}/XCBR1.Pos.stVal", "value": 1},
                                                {"ref": f"{FEED}/XSWI2.Pos.stVal", "value": 1}])
        changed = wait_for(received, lambda event: event["st_num"] > before, 10)
        assert changed, "the batch raised no new GOOSE state"
        time.sleep(2.5)  # retransmissions only from here on: same stNum, rising sqNum
        later = [event["st_num"] for event in frames(received) if event["st_num"] > before]
        assert set(later) == {before + 1}, f"stNum must rise exactly once, saw {sorted(set(later))}"
        # Dbpos 1 (open) of XCBR1 and XSWI2, both in the first frame of the new state.
        assert [value["be"] for value in changed[0]["values"][:2]] == [1, 1], changed[0]["values"]
    finally:
        worker.close()
        listener.close()
        engine.close()
