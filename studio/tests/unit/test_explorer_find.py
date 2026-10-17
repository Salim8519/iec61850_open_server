"""IED Explorer value text, node shape, and Find IEDs range checks and scanning (no network)."""
import time
from pathlib import Path

import pytest
from studio_app.domain.explorer_tree import node_view
from studio_app.domain.explorer_values import describe, quality_text
from studio_app.domain.scan_range import plan_range, suggest_range
from studio_app.tools.explorer import check_endpoint
from studio_app.tools.ied_scan import ScanService


def test_values_are_described_in_plain_words():
    assert describe("LD/XCBR1.Pos.stVal", {"bits": 2, "le": 1, "be": 2})["text"] == "on (closed)"
    assert describe("LD/XCBR1.Pos.stVal", {"bits": 2, "le": 0, "be": 0})["status"] == "warning"
    assert describe("LD/MMXU1.A.q", {"bits": 13, "le": 0, "be": 0}) == {"text": "good", "status": "ok"}
    assert describe("LD/MMXU1.A.q", {"bits": 13, "le": 2 | 64, "be": 0})["text"] == "invalid: failure"
    assert describe("LD/MMXU1.A.q", {"bits": 13, "le": 3 | 2048, "be": 0})["status"] == "warning"
    assert describe("LD/MMXU1.A.t", 1_700_000_000_123)["text"] == "2023-11-14 22:13:20.123 UTC"
    assert describe("LD/MMXU1.A.mag.f", 230.123456)["text"] == "230.123"
    assert describe("LD/CSWI1.Pos.Cancel.T", 0)["text"] == "not set"
    assert describe("LD/X.Beh.stVal", 1)["text"] == "1"
    assert describe("LD/X.Y.z", {"access_error": 10})["status"] == "error"
    assert quality_text(0) == "good"


def test_node_view_groups_attributes_and_blocks():
    answer = {"name": "XCBR1", "leaves": [
        {"ref": "LD/XCBR1.Pos.stVal", "fc": "ST", "mms_type": 3},
        {"ref": "LD/XCBR1.Pos.q", "fc": "ST", "mms_type": 3},
        {"ref": "LD/XCBR1.Pos.ctlModel", "fc": "CF", "mms_type": 4},
        {"ref": "LD/XCBR1.urcb01.RptEna", "fc": "RP", "mms_type": 2}],
        "datasets": ["Events"], "dataset_members": {"Events": {"members": ["LD/XCBR1.Pos[ST]"]}},
        "buffered_reports": [], "unbuffered_reports": ["urcb01"]}
    view = node_view("LD/XCBR1", answer)
    assert [item["name"] for item in view["objects"]] == ["Pos"]
    assert view["objects"][0]["fcs"] == ["ST", "CF"]
    shuffled = dict(answer, leaves=[answer["leaves"][2], answer["leaves"][0]])
    assert [item["fc"] for item in node_view("LD/XCBR1", shuffled)["objects"][0]["attributes"]] == ["ST", "CF"]
    assert view["objects"][0]["attributes"][0] == {"ref": "LD/XCBR1.Pos.stVal", "fc": "ST", "path": "stVal",
                                                   "type": "bit string"}
    assert view["datasets"][0]["members"] == ["LD/XCBR1.Pos[ST]"]
    assert view["reports"] == [{"name": "urcb01", "ref": "LD/XCBR1.RP.urcb01", "buffered": False}]
    assert view["leaf_count"] == 3


def test_endpoint_checks_are_actionable():
    assert check_endpoint(" 10.0.0.5 ", 102) == ("10.0.0.5", 102)
    with pytest.raises(ValueError, match="four numbers"):
        check_endpoint("relay-1", 102)
    with pytest.raises(ValueError, match="between 1 and 65535"):
        check_endpoint("10.0.0.5", 0)


def test_scan_range_is_limited_and_explained():
    assert plan_range("192.168.1.0/24")["count"] == 254
    assert plan_range("192.168.1.7")["hosts"] == ["192.168.1.7"]
    assert plan_range("192.168.1.20-22")["hosts"] == ["192.168.1.20", "192.168.1.21", "192.168.1.22"]
    assert plan_range("192.168.1.0/24", own_ip="192.168.1.1")["count"] == 253
    for bad, reason in (("10.0.0.0/16", "at most 256"), ("hello", "not a valid"), ("10.0.0.9-3", "lower to"),
                        ("224.0.0.1", "Multicast"), ("", "Type a network")):
        with pytest.raises(ValueError, match=reason):
            plan_range(bad)
    assert suggest_range("172.16.5.20", 16) == "172.16.5.0/24"
    assert suggest_range("192.168.1.20", 28) == "192.168.1.16/28"


class FakeWorker:
    def __init__(self, executable):
        self.closed = False

    def request(self, op, **values):
        assert op == "identify"
        if values["ip"].endswith(".3"):
            return {"identified": True, "vendor": "Acme", "model": "R1", "revision": "2", "devices": ["R1CTRL"]}
        return {"identified": False, "reason": "no IEC 61850 association"}

    def close(self):
        self.closed = True


def wait_done(service):
    for _ in range(200):
        if service.status()["state"] != "scanning":
            return service.status()
        time.sleep(0.02)
    raise AssertionError("scan did not finish")


def test_scan_finds_open_ports_and_identifies_them(tmp_path):
    executable = tmp_path / "studio_live.exe"
    executable.write_bytes(b"")
    service = ScanService(executable, probe=lambda host, port: 1.5 if host[-2:] in (".3", ".5") else None,
                          worker=FakeWorker)
    assert service.preview("10.1.1.1-6") == {"range": "10.1.1.1-6", "count": 6, "first": "10.1.1.1",
                                              "last": "10.1.1.6", "port": 102}
    service.start("10.1.1.1-6")
    status = wait_done(service)
    assert status["state"] == "done" and status["probed"] == 6
    found = {entry["ip"]: entry for entry in status["found"]}
    assert set(found) == {"10.1.1.3", "10.1.1.5"}
    assert found["10.1.1.3"]["vendor"] == "Acme" and found["10.1.1.3"]["identified"] is True
    assert found["10.1.1.5"]["identified"] is False and "association" in found["10.1.1.5"]["reason"]
    assert "2 address(es)" in status["message"] and "1 identified" in status["message"]


def test_scan_needs_the_live_client_and_can_be_stopped(tmp_path):
    with pytest.raises(RuntimeError, match="not built"):
        ScanService(Path(tmp_path / "missing.exe")).start("10.1.1.1")
    executable = tmp_path / "studio_live.exe"
    executable.write_bytes(b"")
    slow = ScanService(executable, probe=lambda host, port: time.sleep(0.05), worker=FakeWorker)
    slow.start("10.2.2.0/24")
    status = slow.stop()
    assert status["state"] == "stopped" and status["probed"] <= status["total"]
