"""IED Explorer and Find IEDs operations of studio_live against a studio_ied stand-in."""
import os
import time
from pathlib import Path

import pytest
from studio_app.runtime.worker_process import WorkerProcess
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_live.exe").exists(), reason="Native build required")
IP, PORT = "127.50.0.232", 18132


@pytest.fixture
def device(tmp_path):
    model = import_scl(make_template("ExploreTest"))[0]
    config = tmp_path / "device.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    worker.request("start", cfg=str(config), ip=IP, port=PORT, controls=model["controls"], edition=2)
    yield model, worker
    worker.close()


@pytest.fixture
def live():
    process = WorkerProcess(BIN / "studio_live.exe", lambda event: None)
    yield process
    process.close()


def test_explore_one_level_at_a_time(device, live):
    live.request("connect", ip=IP, port=PORT)
    devices = live.request("explore")["names"]
    assert "ExploreTestCTRL" in devices
    nodes = live.request("explore", device="ExploreTestCTRL")["names"]
    assert {"LLN0", "XCBR1", "GGIO1"} <= set(nodes)
    node = live.request("explore", node="ExploreTestCTRL/GGIO1")
    assert node["name"] == "GGIO1"
    leaves = {(leaf["ref"], leaf["fc"]) for leaf in node["leaves"]}
    assert ("ExploreTestCTRL/GGIO1.AnIn1.mag.f", "MX") in leaves
    llno = live.request("explore", node="ExploreTestCTRL/LLN0")
    assert llno["datasets"] and llno["dataset_members"]


def test_explore_needs_a_connection(live):
    with pytest.raises(RuntimeError, match="Not connected"):
        live.request("explore")


def test_identify_reports_identity_and_devices(device, live):
    answer = live.request("identify", ip=IP, port=PORT, timeout_ms=3000)
    assert answer["identified"] is True
    assert "ExploreTestCTRL" in answer["devices"]
    assert {"vendor", "model", "revision", "connect_ms"} <= set(answer)
    assert (answer["vendor"], answer["model"]) == ("ScadaBench", "Simulated IED"), "not the old product name"


def test_identify_unreachable_port_gives_a_reason(live):
    answer = live.request("identify", ip="127.50.0.233", port=18199, timeout_ms=800)
    assert answer["identified"] is False and answer["reason"]


def test_explorer_service_browses_and_polls_a_watch_list(device):
    from studio_app.tools.explorer import ExplorerService
    _, worker = device
    service = ExplorerService(BIN / "studio_live.exe")
    try:
        assert service.connect(IP, PORT)["state"] == "connected"
        assert "ExploreTestCTRL" in service.devices()
        node = service.node("ExploreTestCTRL/GGIO1")
        attribute = next(item for obj in node["objects"] if obj["name"] == "AnIn1"
                         for item in obj["attributes"] if item["path"] == "mag.f")
        worker.request("update", ref="ExploreTestCTRL/GGIO1.AnIn1.mag.f", value=12.5)
        service.set_watch([attribute, {"ref": "ExploreTestCTRL/XCBR1.Pos.stVal", "fc": "ST"}], interval_ms=500)
        service.start_polling()
        deadline = time.time() + 5
        while time.time() < deadline and service.status()["reads"] < 2:
            time.sleep(0.05)
        watch = {item["ref"]: item for item in service.status()["watch"]}
        assert watch["ExploreTestCTRL/GGIO1.AnIn1.mag.f"]["text"] == "12.5"
        assert watch["ExploreTestCTRL/XCBR1.Pos.stVal"]["status"] in ("ok", "warning")
        worker.request("update", ref="ExploreTestCTRL/GGIO1.AnIn1.mag.f", value=99.0)
        reads = service.status()["reads"]
        while time.time() < deadline + 3 and service.status()["reads"] < reads + 2:
            time.sleep(0.05)
        assert {item["ref"]: item for item in service.status()["watch"]}["ExploreTestCTRL/GGIO1.AnIn1.mag.f"]["text"] == "99"
    finally:
        assert service.disconnect()["state"] == "disconnected"
        service.close()


def test_find_ieds_scans_a_range_and_identifies_the_stand_in(device):
    from studio_app.tools.ied_scan import ScanService
    service = ScanService(BIN / "studio_live.exe")
    service.start("127.50.0.230-234", port=PORT)
    deadline = time.time() + 20
    while time.time() < deadline and service.status()["state"] == "scanning":
        time.sleep(0.05)
    status = service.status()
    assert status["state"] == "done", status
    assert [entry["ip"] for entry in status["found"]] == [IP]
    assert status["found"][0]["identified"] is True and "ExploreTestCTRL" in status["found"][0]["devices"]


def test_explorer_reconnects_after_overlapping_and_failed_connects(device):
    import threading
    from studio_app.tools.explorer import ExplorerService
    service = ExplorerService(BIN / "studio_live.exe")
    try:
        threads = [threading.Thread(target=service.connect, args=(IP, PORT)) for _ in range(2)]
        for thread in threads:
            thread.start()
        for thread in threads:
            thread.join(20)
        assert service.status()["state"] == "connected"
        service.state = "failed"  # the worker is still associated, as an overlapping connect used to leave it
        assert service.connect(IP, PORT)["state"] == "connected"
        assert "ExploreTestCTRL" in service.devices()
    finally:
        service.disconnect()
        service.close()
