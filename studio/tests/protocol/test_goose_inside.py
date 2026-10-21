"""GOOSE inside the app (ADR-0073): studio_ied -> studio_goose engine -> studio_live, no adapter.

GOOSE-05: workers publish only to the engine. GOOSE-11: the simulated publisher reaches
the decoder through the engine. GOOSE-17: none of this needs Npcap or an adapter.
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


def wait_for(events, predicate, seconds):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        found = [event for event in list(events) if predicate(event)]
        if found:
            return found
        time.sleep(0.05)
    return []


def subscription(control):
    return {"key": control["gocb_ref"], "gocb_ref": control["gocb_ref"], "app_id": control["app_id"],
            "mac": control["mac"], "go_id": control["go_id"], "dataset_ref": control["dataset_ref"],
            "conf_rev": control["conf_rev"]}


@pytest.fixture
def publisher(tmp_path):
    model = import_scl((ROOT / "scd/open_substation.scd").read_bytes(), "open_substation.scd", ["IED1_XCBR"])[0]
    config = tmp_path / "publisher.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    return model, config


def test_simulated_publisher_reaches_the_decoder_through_the_engine(publisher):
    model, config = publisher
    control = model["goose"]["publishes"][0]
    received, engine_events = [], []
    engine = WorkerProcess(BIN / "studio_goose.exe", engine_events.append)
    listener = WorkerProcess(BIN / "studio_live.exe", received.append)
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    try:
        assert "goose_engine_v1" in engine.request("hello")["capabilities"]
        opened = engine.request("open", ieds=[{"id": "IED1_XCBR", "gocb_refs": [control["gocb_ref"]]}])
        mac = opened["sources"]["IED1_XCBR"]
        inbox = listener.request("goose_start", timeout=30, subscriptions=[subscription(control)])["inbox_port"]
        engine.request("deliver", port=inbox)
        started = worker.request("start", cfg=str(config), ip="127.50.0.233", port=18133, controls=model["controls"],
                                 edition=2, goose={"publish": True, "interface": f"udp:{opened['port']}:{mac.replace('-', '')}"})
        assert started["goose"]["active"] is True and "goose_bus" in started["capabilities"]
        first = wait_for(received, lambda e: e.get("event") == "goose_rx" and e.get("state") == "receiving", 20)
        assert first, "No GOOSE frame reached studio_live through the engine"
        frame = first[0]
        assert frame["origin"] == "app" and frame["src_mac"] == mac
        assert frame["conf_rev"] == control["conf_rev"] and frame["dataset"] == control["dataset_ref"]
        assert frame["problem"] == ""
        before = frame["st_num"]
        worker.request("update", ref="IED1_XCBRGenericIO/XCBR1.Pos.stVal", value=2)
        changed = wait_for(received, lambda e: e.get("event") == "goose_rx" and (e.get("st_num") or 0) > before, 10)
        assert changed and changed[0]["values"] == [{"bits": 2, "le": 1, "be": 2}]
        stats = engine.request("stats")
        assert stats["link"]["adapter_guid"] == "" and stats["sent"] == 0  # GOOSE-17: no adapter, nothing out
        assert stats["delivered_app"] >= 2 and stats["bus_rejected"] == 0
    finally:
        worker.close()
        listener.close()
        engine.close()


@pytest.mark.parametrize("goose", [{"publish": True, "adapter_guid": "{00000000-0000-0000-0000-000000000000}"},
                                   {"publish": True, "interface": "3"}])
def test_worker_refuses_any_interface_but_the_engine(publisher, goose):
    model, config = publisher
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    try:
        started = worker.request("start", cfg=str(config), ip="127.50.0.232", port=18132,
                                 controls=model["controls"], edition=2, goose=goose)
        assert started["goose"]["requested"] is True and started["goose"]["active"] is False
        assert "GOOSE engine" in started["goose"]["reason"]
    finally:
        worker.close()
