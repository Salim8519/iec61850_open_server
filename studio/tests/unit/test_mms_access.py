"""MMS client access is summarised per client, never logged per read (MMSACC-01..04)."""
from pathlib import Path

from studio_app.domain.mms_access import client_view, merge_summary
from studio_app.simulation.mms_access import MmsAccess

NATIVE = Path(__file__).resolve().parents[2] / "native/worker"


def summary(peer, reads=None, connected=True, rcbs=(), writes=None, total=0):
    return {"peer": peer, "connected": connected, "reads": reads or {}, "writes": writes or {},
            "rcbs": list(rcbs), "total_reads": total}


def test_reads_are_counted_per_logical_node_and_clients_by_address():
    clients = {}
    merge_summary(clients, [summary("10.0.0.9:50001", {"IED1LD0/XCBR1": 3, "IED1LD0/LLN0": 1}, total=4)], 100)
    merge_summary(clients, [summary("10.0.0.9:50001", {"IED1LD0/XCBR1": 2}, rcbs=["LLN0.brcb01"], total=6)], 101)
    row = client_view(clients, 102)[0]
    assert row["ip"] == "10.0.0.9" and row["reading"] and row["total_reads"] == 6
    assert row["reads"][0] == {"node": "IED1LD0/XCBR1", "count": 5} and row["rcbs"] == ["LLN0.brcb01"]
    assert client_view(clients, 120)[0]["reading"] is False


def test_a_client_that_disconnects_is_removed():
    clients = {}
    merge_summary(clients, [summary("10.0.0.9:50001", {"A/B": 1})], 1)
    merge_summary(clients, [summary("10.0.0.9:50001", connected=False)], 2)
    assert clients == {}


class Bus:
    def __init__(self):
        self.events = []

    def publish(self, kind, **payload):
        self.events.append((kind, payload))


class Coordinator:
    def __init__(self):
        self.events = Bus()


def test_scadabench_own_command_is_not_an_external_control():
    tracker = MmsAccess(Coordinator())
    tracker.mark_own("ied", "LD0/CSWI1.Pos")
    assert tracker.control("ied", {"ref": "LD0/CSWI1.Pos", "peer": "127.0.0.1:6000"}) is False
    assert tracker.snapshot()["ied"][0]["own"] is True
    assert tracker.control("ied", {"ref": "LD0/CSWI1.Pos", "peer": "10.0.0.9:6001"}) is True


def test_summaries_go_to_the_live_view_only():
    coordinator = Coordinator()
    MmsAccess(coordinator).update("ied", {"clients": [summary("10.0.0.9:1", {"A/B": 1})]})
    assert [kind for kind, _ in coordinator.events.events] == ["mms_access"]
    source = (Path(__file__).resolve().parents[2] / "backend/studio_app/simulation/coordinator.py").read_text()
    assert 'if kind == "mms_access":' in source and "self.mms_access.update(ied_id, event)\n            return" in source


def test_worker_observes_reads_without_changing_the_answer():
    hook = (NATIVE / "server/read_hook.c").read_text()
    assert "hook->original(hook->original_parameter" in hook and "return result;" in hook
    assert "IedServer_setReadAccessHandler" not in (NATIVE / "server/server.cpp").read_text()
    log = (NATIVE / "server/access_log.cpp").read_text()
    assert "milliseconds(1000)" in log and '"mms_access"' in log
