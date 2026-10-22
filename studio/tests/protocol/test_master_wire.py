"""ScadaBench as SCADA master (studio_master) against a 104 controlled station and a DNP3
outstation on the wire. The device here is studio_gateway; independent outstations are covered in
test_master_independent.py. Spec: docs/specs/scada-master.md."""
import threading
import time

import pytest
from studio_app.runtime.worker_process import WorkerProcess
from test_gateway_wire import BIN, POINTS, Gateway

pytestmark = pytest.mark.skipif(not (BIN / "studio_master.exe").exists(), reason="Master build required")


class Master:
    def __init__(self):
        self.values, self.links, self.traffic, self.finished = [], [], [], threading.Event()
        self.process = WorkerProcess(BIN / "studio_master.exe", self.event)

    def event(self, event):
        kind = event.get("event")
        if kind == "master_values":
            self.values.extend(event["values"])
        elif kind == "master_link":
            self.links.append(event["state"])
        elif kind == "master_interrogation":
            self.finished.set()
        elif kind == "gw_traffic":
            self.traffic.append(event["text"])

    def latest(self, address):
        found = [value for value in self.values if value["address"] == address]
        return found[-1] if found else None

    def wait_for(self, check, timeout=5):
        end = time.time() + timeout
        while time.time() < end and not check():
            time.sleep(0.05)
        return check()

    def close(self):
        self.process.close()


@pytest.fixture
def station():
    gateway = Gateway("iec104", {"bind": "127.0.0.1", "port": 12410, "common_address": 7})
    master = Master()
    yield gateway, master
    master.close()
    gateway.close()


def test_master_discovers_every_point_by_general_interrogation(station):
    gateway, master = station
    opened = master.process.request("connect", protocol="iec104",
                                    settings={"host": "127.0.0.1", "port": 12410, "common_address": 7})
    assert opened["common_address"] == 7 and master.wait_for(lambda: "open" in master.links)
    assert master.process.request("interrogate")["confirmed"] is True
    assert master.finished.wait(10), "no activation termination after the interrogation"
    found = {value["address"]: value for value in master.values if value["cause"] == "interrogated"}
    assert found[1001]["group"] == "single" and found[1001]["value"] == 1
    assert found[2001]["group"] == "double" and found[2001]["value"] == 2
    assert found[3001]["group"] == "float" and found[3001]["value"] == pytest.approx(230.5)
    assert found[4001]["quality"] & 0x80  # invalid in IEC 61850 arrives as IV
    gateway.process.request("update", values=[{"id": "mv", "value": 231.0, "quality": 0}])
    assert master.wait_for(lambda: (master.latest(3001) or {}).get("cause") == "spontaneous")
    assert master.latest(3001)["value"] == pytest.approx(231.0)


def test_master_commands_get_confirmation_termination_and_refusals(station):
    gateway, master = station
    master.process.request("connect", protocol="iec104", settings={"host": "127.0.0.1", "port": 12410,
                                                                    "common_address": 7})
    master.wait_for(lambda: "open" in master.links)
    done = master.process.request("command", kind="double", address=6001, value=2, select=True)
    assert done["ok"] and [step["step"] for step in done["steps"]] == ["select", "execute", "termination"]
    assert done["terminated"] and gateway.commands == [("cb", True)]
    refused = master.process.request("command", kind="single", address=6002, value=1)
    assert not refused["ok"] and "refused" in refused["steps"][0]["text"]
    unknown = master.process.request("command", kind="double", address=6999, value=1)
    assert not unknown["ok"] and "unknown object address" in unknown["steps"][0]["text"]
    with pytest.raises(RuntimeError, match="1 \\(OFF / open\\) or 2"):
        master.process.request("command", kind="double", address=6001, value=3)
    assert any("C_DC_NA_1 execute IOA 6001" in line for line in master.traffic)


def test_master_explains_an_unreachable_device_and_disconnects_cleanly():
    master = Master()
    try:
        # 0178 (C15): a refused connect said "No answer ... within t0" although nobody waited.
        with pytest.raises(RuntimeError, match="Nothing is listening on 127.0.0.1:12419"):
            master.process.request("connect", protocol="iec104",
                                   settings={"host": "127.0.0.1", "port": 12419, "t0": 2})
        with pytest.raises(RuntimeError, match="press Connect first"):
            master.process.request("interrogate")
        with pytest.raises(RuntimeError, match="not available as master"):
            master.process.request("connect", protocol="iec101", settings={"host": "127.0.0.1"})
        assert master.process.request("disconnect") is True
    finally:
        master.close()


def test_dnp3_master_integrity_poll_events_and_crob():
    points = [dict(point, address=index) for index, point in enumerate(POINTS)]
    gateway = Gateway("dnp3", {"bind": "127.0.0.1", "port": 20211, "outstation_address": 10, "master_address": 1}, points)
    master = Master()
    try:
        opened = master.process.request("connect", timeout=20, protocol="dnp3", settings={
            "host": "127.0.0.1", "port": 20211, "outstation_address": 10, "master_address": 1, "event_seconds": 1})
        assert opened["outstation_address"] == 10
        assert master.process.request("interrogate")["confirmed"] is True
        assert master.finished.wait(10), "the integrity poll did not finish"
        found = {(value["group"], value["address"]): value for value in master.values if value["cause"] == "integrity poll"}
        assert found[("binary", 0)]["value"] == 1 and found[("binary", 0)]["quality"] & 1
        assert found[("double_binary", 1)]["value"] == 2
        assert found[("analog", 2)]["value"] == pytest.approx(230.5)
        assert not found[("analog", 3)]["quality"] & 1  # invalid in IEC 61850 -> not ONLINE
        gateway.process.request("update", values=[{"id": "mv", "value": 231.0, "quality": 0}])
        assert master.wait_for(lambda: any(v["address"] == 2 and v["cause"] in ("event", "unsolicited")
                                           and v["value"] == pytest.approx(231.0) for v in master.values))
        done = master.process.request("command", timeout=20, kind="binary_output", address=4, value=1, select=True)
        assert done["ok"] and done["steps"][0]["step"] == "select and operate" and gateway.commands == [("cb", True)]
        refused = master.process.request("command", timeout=20, kind="binary_output", address=5, value=1)
        assert not refused["ok"] and "not authorized" in refused["steps"][0]["text"]
        missing = master.process.request("command", timeout=20, kind="binary_output", address=99, value=1)
        text = missing["steps"][0]["text"]  # change 0089: was "refused: not supported (success)"
        assert not missing["ok"] and "no such control point" in text and "success" not in text
        assert any("CROB latch on index 4" in line for line in master.traffic)
    finally:
        master.close()
        gateway.close()


def test_dnp3_master_explains_an_unreachable_outstation():
    master = Master()
    try:
        with pytest.raises(RuntimeError, match="Nothing is listening on 127.0.0.1:20219"):
            master.process.request("connect", timeout=20, protocol="dnp3", settings={"host": "127.0.0.1", "port": 20219})
        with pytest.raises(RuntimeError, match="Nothing is listening on 127.0.0.1:15219"):
            master.process.request("connect", timeout=20, protocol="modbus", settings={"host": "127.0.0.1", "port": 15219})
    finally:
        master.close()
