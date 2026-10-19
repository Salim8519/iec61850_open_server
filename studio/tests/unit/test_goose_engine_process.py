"""The backend's side of the GOOSE engine: stale binaries, bus interfaces, events (GOOSE-27)."""
from pathlib import Path

import pytest
from studio_app.acquisition.goose_engine import GooseEngine, publishers
from studio_app.runtime.supervisor import Supervisor

PUBLISHER = {"id": "P", "role": "simulated", "goose_publish": True,
             "goose": {"publishes": [{"gocb_ref": "P/LLN0$GO$g"}]}}


class Process:
    def __init__(self, capabilities):
        self.capabilities = capabilities
        self.requests = []
        self.closed = False

    def request(self, op, **values):
        self.requests.append((op, values))
        if op == "hello":
            return {"capabilities": self.capabilities}
        if op == "open":
            return {"port": 50123, "sources": {item["id"]: "02-5B-12-00-00-01" for item in values["ieds"]}}
        return {}

    def close(self):
        self.closed = True


class Manager:
    def __init__(self, folder):
        self.executable = Path(folder) / "studio_live.exe"
        self.events = []
        self.coordinator = self

    def emit(self, kind, **payload):
        self.events.append((kind, payload))


def engine(tmp_path, capabilities=("goose_engine_v1",)):
    (tmp_path / "studio_goose.exe").write_bytes(b"")
    process = Process(list(capabilities))
    return GooseEngine(Manager(tmp_path), starter=lambda exe, on_event: process), process


def test_missing_or_stale_engine_is_refused(tmp_path):
    with pytest.raises(ValueError, match="not built"):
        GooseEngine(Manager(tmp_path)).ensure()
    stale, process = engine(tmp_path, capabilities=())
    with pytest.raises(ValueError, match="outdated"):
        stale.ensure()
    assert process.closed and not stale.running


def test_bus_for_gives_each_publishing_simulated_ied_its_engine_interface(tmp_path):
    goose, process = engine(tmp_path)
    off = {**PUBLISHER, "id": "Q", "goose_publish": False}
    real = {**PUBLISHER, "id": "R", "role": "real"}
    assert publishers([PUBLISHER, off, real]) == [{"id": "P", "gocb_refs": ["P/LLN0$GO$g"]}]
    assert goose.bus_for([PUBLISHER, off, real]) == {"P": "udp:50123:025B12000001"}
    assert goose.bus_for([off]) == {}


def test_conflict_share_and_uplink_events_reach_the_ui_and_snapshot(tmp_path):
    goose, _ = engine(tmp_path)
    goose.ensure()
    goose._event({"event": "goose_conflict", "ied_id": "P", "gocb_ref": "P/LLN0$GO$g",
                  "src_mac": "00-30-A7-10-20-30", "sharing": True})
    goose._event({"event": "goose_share", "state": "sharing"})
    goose._event({"event": "goose_uplink", "state": "send_failed", "message": "cable unplugged"})
    snapshot = goose.snapshot()
    assert snapshot["share"] == "sharing" and snapshot["uplink"] == "send_failed"
    assert snapshot["conflicts"][0]["src_mac"] == "00-30-A7-10-20-30"
    kinds = [kind for kind, _ in goose.manager.events]
    assert kinds == ["goose_conflict", "goose_share", "goose_uplink"]


def test_supervisor_asks_the_engine_before_any_worker_starts(tmp_path):
    order = []

    class Recorder(Supervisor):
        def _launch(self, ied, network, bus=""):
            order.append(("launch", ied["id"], bus))

    (tmp_path / "studio_ied.exe").write_bytes(b"")
    supervisor = Recorder(tmp_path / "studio_ied.exe", tmp_path,
                          on_event=lambda *a: None, lab_addresses=lambda guid: set(),
                          goose_bus=lambda ieds: order.append(("bus", [i["id"] for i in ieds])) or {"P": "udp:1:02"})
    ied = {**PUBLISHER, "name": "P", "ip": "127.50.0.2", "port": 102, "cfg": "", "controls": [], "signals": []}
    supervisor.start([ied], {"mode": "loopback", "adapter_guid": ""})
    assert order == [("bus", ["P"]), ("launch", "P", "udp:1:02")]


def test_events_arriving_before_a_reply_never_deadlock(tmp_path):
    """The reader thread delivers events while a request waits (Run froze: uplink timed out)."""
    import threading
    goose, process = engine(tmp_path)
    goose.ensure()
    delivered = []

    def request(op, **values):
        if op == "uplink":  # like WorkerProcess: the event is read on another thread before the reply
            reader = threading.Thread(target=lambda: delivered.append(
                goose._event({"event": "goose_share", "state": "checking"}) or True))
            reader.start()
            reader.join(timeout=2)
            assert delivered, "the event reader was blocked by the waiting request"
            return {"adapter_guid": values["adapter_guid"], "share": "checking", "uplink": "open"}
        return {}

    process.request = request
    assert goose.uplink("{15A6105F-A715-444F-9D7C-AA259A2EED8A}", True)["share"] == "checking"
