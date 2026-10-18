"""Gateway simulator against independent masters: lib60870 (IEC 104), opendnp3 (DNP3) and
a raw-socket Modbus TCP client written here. Spec: docs/specs/gateway.md."""
import json
import os
import socket
import struct
import subprocess
from pathlib import Path

import pytest
from studio_app.runtime.worker_process import WorkerProcess

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_gateway.exe").exists(), reason="Gateway build required")
POINTS = [
    {"id": "sp", "kind": "single", "address": 1001},
    {"id": "dp", "kind": "double", "address": 2001},
    {"id": "mv", "kind": "float", "address": 3001},
    {"id": "iv", "kind": "integer", "address": 4001},
    {"id": "cb", "kind": "command_double", "address": 6001},
    {"id": "real", "kind": "command_single", "address": 6002, "monitor_only": True},
]
VALUES = [{"id": "sp", "value": True, "quality": 0}, {"id": "dp", "value": 2, "quality": 0},
          {"id": "mv", "value": 230.5, "quality": 0}, {"id": "iv", "value": 42, "quality": 2}]


class Gateway:
    def __init__(self, protocol, settings, points=POINTS):
        self.commands = []
        self.traffic = []
        self.process = WorkerProcess(BIN / "studio_gateway.exe", self.event)
        try:
            self.opened = self.process.request("start", protocol=protocol, settings=settings, points=points)
        except Exception:
            self.process.close()
            raise
        self.process.request("update", values=VALUES)

    def event(self, event):
        if event.get("event") == "gw_command":
            self.commands.append((event["point_id"], event["value"]))
            self.process.request("command_result", token=event["token"], ok=True)
        elif event.get("event") == "gw_traffic":
            self.traffic.append(event["text"])

    def close(self):
        self.process.close()


def run(tool, *args):
    done = subprocess.run([str(BIN / tool), *map(str, args)], capture_output=True, text=True, timeout=60)
    return [json.loads(line) for line in done.stdout.splitlines() if line.startswith("{")]


def test_iec104_interrogation_commands_and_refusals():
    gateway = Gateway("iec104", {"bind": "127.0.0.1", "port": 12404, "common_address": 7})
    try:
        frames = run("studio_test_iec104_master.exe", "127.0.0.1", 12404, 7, "gi", "dc:6001:2:select", "dc:6001:2",
                     "sc:6002:1", "sc:6001:1", "dc:6999:1", "read:3001")
        objects = {item["ioa"]: item for frame in frames if frame["cot"] == 20 for item in frame["objects"]}
        assert objects[1001]["value"] == 1 and objects[2001]["value"] == 2
        assert objects[3001]["value"] == pytest.approx(230.5) and objects[4001]["value"] == 42
        assert objects[4001]["quality"] & 0x80  # IEC 61850 invalid -> 104 IV
        commands = [frame for frame in frames if frame["type"] in (45, 46)]
        assert [(frame["cot"], frame["negative"]) for frame in commands] == [
            (7, False),           # select confirmed
            (7, False), (10, False),  # execute confirmed and terminated
            (7, True),            # real IED point: negative confirmation
            (44, True),           # single command to a double-command IOA
            (47, True)]           # unknown IOA
        assert gateway.commands == [("cb", True)]
        assert any(frame["cot"] == 5 and frame["objects"][0]["ioa"] == 3001 for frame in frames)
    finally:
        gateway.close()


def test_iec104_real_ied_refusal_names_the_owner_switch():
    """Change 0089: the log said real IEDs are "never" commanded, but the owner's switch decides."""
    gateway = Gateway("iec104", {"bind": "127.0.0.1", "port": 12405, "common_address": 7})
    try:
        frames = run("studio_test_iec104_master.exe", "127.0.0.1", 12405, 7, "sc:6002:1")
        assert [(frame["cot"], frame["negative"]) for frame in frames if frame["type"] == 45] == [(7, True)]
        assert any("commands to real IEDs are off in the gateway settings" in line for line in gateway.traffic)
    finally:
        gateway.close()


def test_iec104_spontaneous_change_and_wrong_common_address():
    gateway = Gateway("iec104", {"bind": "127.0.0.1", "port": 12405, "common_address": 1})
    try:
        master = subprocess.Popen([str(BIN / "studio_test_iec104_master.exe"), "127.0.0.1", "12405", "1", "wait:1500"],
                                  stdout=subprocess.PIPE, text=True)
        import time
        time.sleep(0.8)
        gateway.process.request("update", values=[{"id": "mv", "value": 231.25, "quality": 0, "time_ms": 1_700_000_000_000}])
        output, _ = master.communicate(timeout=20)
        frames = [json.loads(line) for line in output.splitlines() if line.startswith("{")]
        spont = [frame for frame in frames if frame["cot"] == 3]
        assert spont and spont[0]["type"] == 36 and spont[0]["objects"][0]["value"] == pytest.approx(231.25)
        wrong = run("studio_test_iec104_master.exe", "127.0.0.1", 12405, 9, "gi")
        assert any(frame["cot"] == 46 and frame["negative"] for frame in wrong)
    finally:
        gateway.close()


def modbus(sock, unit, pdu, transaction=1):
    sock.sendall(struct.pack(">HHHB", transaction, 0, len(pdu) + 1, unit) + pdu)
    header = sock.recv(7)
    length = struct.unpack(">H", header[4:6])[0]
    return sock.recv(length - 1)


def test_modbus_tcp_reads_and_coil_commands():
    points = [dict(point, address=index) for index, point in enumerate(POINTS[:2])] + [
        {"id": "mv", "kind": "float", "address": 10}, {"id": "iv", "kind": "integer", "address": 12},
        {"id": "cb", "kind": "command_double", "address": 0}, {"id": "real", "kind": "command_single", "address": 1,
                                                              "monitor_only": True}]
    points[1]["address"] = 5
    gateway = Gateway("modbus", {"bind": "127.0.0.1", "port": 15020, "unit_id": 3}, points)
    try:
        with socket.create_connection(("127.0.0.1", 15020), timeout=5) as sock:
            assert modbus(sock, 3, bytes([2, 0, 0, 0, 1])) == bytes([2, 1, 1])  # discrete input 0 = single ON
            registers = modbus(sock, 3, bytes([4, 0, 5, 0, 1]))
            assert registers == bytes([4, 2, 0, 2])  # double point ON = 2
            float_regs = modbus(sock, 3, bytes([3, 0, 10, 0, 4]))
            assert struct.unpack(">f", float_regs[2:6])[0] == pytest.approx(230.5)
            assert struct.unpack(">i", float_regs[6:10])[0] == 42
            assert modbus(sock, 3, bytes([5, 0, 0, 0xFF, 0])) == bytes([5, 0, 0, 0xFF, 0])
            assert modbus(sock, 3, bytes([5, 0, 1, 0xFF, 0])) == bytes([0x85, 1])  # real IED refused
            assert modbus(sock, 3, bytes([5, 0, 9, 0xFF, 0])) == bytes([0x85, 2])  # no command there
            assert modbus(sock, 3, bytes([1, 0, 0, 0, 1])) == bytes([1, 1, 1])  # coil keeps last command
            assert modbus(sock, 3, bytes([6, 0, 0, 0, 1])) == bytes([0x86, 1])  # writes to registers refused
            assert modbus(sock, 9, bytes([3, 0, 10, 0, 2])) == bytes([0x83, 0x0B])  # other unit id
        assert gateway.commands == [("cb", True)]
    finally:
        gateway.close()


def test_dnp3_integrity_scan_and_crob():
    points = [dict(point, address=index) for index, point in enumerate(POINTS)]
    gateway = Gateway("dnp3", {"bind": "127.0.0.1", "port": 20201, "outstation_address": 10, "master_address": 1}, points)
    try:
        lines = run("studio_test_dnp3_master.exe", "127.0.0.1", 20201, 1, 10, "scan", "crob:4:on", "crob:5:on", "crob:9:off")
        values = {(item["kind"], item["index"]): item for item in lines if "kind" in item}
        assert values[("binary", 0)]["value"] == 1 and values[("binary", 0)]["flags"] & 1
        assert values[("double", 1)]["value"] == 2
        assert values[("analog", 2)]["value"] == pytest.approx(230.5)
        assert not values[("analog", 3)]["flags"] & 1  # invalid quality -> not ONLINE
        results = {item["command"]: item["status"] for item in lines if "command" in item}
        assert results == {4: "SUCCESS", 5: "NOT_AUTHORIZED", 9: "NOT_SUPPORTED"}
        assert gateway.commands == [("cb", True)]
    finally:
        gateway.close()


def test_listen_failure_is_explained():
    first = Gateway("modbus", {"bind": "127.0.0.1", "port": 15021})
    try:
        with pytest.raises(RuntimeError, match="Could not listen"):
            Gateway("modbus", {"bind": "127.0.0.1", "port": 15021})
    finally:
        first.close()
