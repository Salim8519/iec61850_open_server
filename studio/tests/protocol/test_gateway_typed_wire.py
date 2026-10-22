"""Typed gateway commands (ADR-0107) against independent masters: lib60870 (IEC 104 step,
set points, select/execute/cancel), opendnp3 (CROB codes, SBO, direct operate, g41) and raw
Modbus TCP (FC6/FC16 set points). The fake engine records what the outstation hands over;
the engine's routing is tested in Go (internal/gateway/service/routing_test.go)."""
import socket
import struct
import threading

import pytest
from studio_app.runtime.worker_process import WorkerProcess
from test_gateway_wire import BIN, modbus, run

pytestmark = pytest.mark.skipif(not (BIN / "studio_gateway.exe").exists(), reason="Gateway build required")


class TypedGateway:
    """studio_gateway started with typed_commands; the engine refuses a set point of 5000 or 999."""

    def __init__(self, protocol, settings, points, values=()):
        self.commands = []
        self.traffic = []
        self.process = WorkerProcess(BIN / "studio_gateway.exe", self.event)
        hello = self.process.request("hello")
        assert "typed_commands" in hello["capabilities"]
        try:
            self.process.request("start", protocol=protocol, settings=settings, points=points, typed_commands=True)
        except Exception:
            self.process.close()
            raise
        if values:
            self.process.request("update", values=list(values))

    def event(self, event):
        if event.get("event") == "gw_command":
            self.commands.append((event["point_id"], event["value"], event.get("wire")))
            refused = event["value"] in (5000, 999)
            # Answer from another thread: events arrive on the worker's reader thread, which must
            # stay free to read the answer to this request.
            threading.Thread(target=self.process.request, args=("command_result",), daemon=True,
                             kwargs={"token": event["token"], "ok": not refused,
                                     "reason": "the IED refused the value" if refused else ""}).start()
        elif event.get("event") == "gw_traffic":
            self.traffic.append(event["text"])

    def close(self):
        self.process.close()


IEC104_POINTS = [
    {"id": "tp", "kind": "step_position", "address": 4501},
    {"id": "cb", "kind": "command_double", "address": 6001},
    {"id": "tap", "kind": "command_step", "address": 6003},
    {"id": "sn", "kind": "command_setpoint", "address": 6004, "format": "normalized"},
    {"id": "ss", "kind": "command_setpoint", "address": 6005, "format": "scaled"},
    {"id": "sf", "kind": "command_setpoint", "address": 6006, "format": "float"},
    {"id": "real", "kind": "command_step", "address": 6007, "monitor_only": True},
]


def test_iec104_step_set_points_select_execute_cancel():
    gateway = TypedGateway("iec104", {"bind": "127.0.0.1", "port": 12414, "common_address": 7}, IEC104_POINTS,
                           [{"id": "tp", "value": 5, "quality": 0}])
    try:
        frames = run("studio_test_iec104_master.exe", "127.0.0.1", 12414, 7, "gi",
                     "rc:6003:2:select", "rc:6003:2", "rc:6003:1:cancel", "sen:6004:0.5", "seb:6005:120:select",
                     "seb:6005:120", "sec:6006:12.5:time", "sec:6006:5000", "sec:6005:1", "rc:6999:1", "rc:6007:2",
                     "dc:6001:2:select", "dc:6001:2", "dc:6001:2:cancel")
        steps = [frame for frame in frames if 45 <= frame["type"] <= 63]
        assert [(frame["type"], frame["cot"], frame["negative"]) for frame in steps] == [
            (47, 7, False),                  # step select confirmed
            (47, 7, False), (47, 10, False),  # step execute: confirmation and termination
            (47, 9, False),                  # cancel (COT 8) confirmed
            (48, 7, False), (48, 10, False),  # normalized set point
            (49, 7, False),                  # scaled set point select
            (49, 7, False), (49, 10, False),
            (63, 7, False), (63, 10, False),  # float set point with time tag (C_SE_TC_1)
            (50, 7, True),                   # the engine refused the value: negative confirmation
            (50, 44, True),                  # float set point to a scaled set point: wrong type
            (47, 47, True),                  # unknown IOA
            (47, 7, True),                   # real IED while real commands are off
            (46, 7, False),                  # double command select now goes to the engine too
            (46, 7, False), (46, 10, False),
            (46, 9, False)]
        tap = [item for item in frames if item["cot"] == 20 for item in item["objects"] if item["ioa"] == 4501]
        assert tap and tap[0]["value"] == 5
        assert any(frame["type"] == 5 and frame["cot"] == 20 for frame in frames)  # M_ST_NA_1 in the GI
        handed = [(point, value, wire["select"], wire["cot"], wire["type_id"]) for point, value, wire in gateway.commands]
        assert handed[:9] == [
            ("tap", 2, True, 6, 47), ("tap", 2, False, 6, 47), ("tap", 1, False, 8, 47),
            ("sn", pytest.approx(0.5, abs=1e-4), False, 6, 48), ("ss", 120, True, 6, 49), ("ss", 120, False, 6, 49),
            ("sf", 12.5, False, 6, 63), ("sf", 5000, False, 6, 50), ("cb", True, True, 6, 46)]
        assert handed[9:] == [("cb", True, False, 6, 46), ("cb", True, False, 8, 46)]
        assert any("expects a scaled set point" in line for line in gateway.traffic)
        assert any("no command at this IOA" in line for line in gateway.traffic)
    finally:
        gateway.close()


def test_iec104_without_typed_commands_keeps_local_select_and_ignores_new_types():
    """The Python-era start request: select/cancel confirmed by the outstation, C_RC unknown."""
    points = [{"id": "cb", "kind": "command_double", "address": 6001}]
    process = WorkerProcess(BIN / "studio_gateway.exe", lambda event: None)
    try:
        process.request("start", protocol="iec104", settings={"bind": "127.0.0.1", "port": 12415}, points=points)
        frames = run("studio_test_iec104_master.exe", "127.0.0.1", 12415, 1, "dc:6001:2:select", "dc:6001:2:cancel",
                     "rc:6001:2")
        assert [(frame["type"], frame["cot"], frame["negative"]) for frame in frames if 45 <= frame["type"] <= 63] == [
            (46, 7, False), (46, 9, False), (47, 44, True)]
    finally:
        process.close()


def test_dnp3_crob_codes_sbo_direct_and_analog_outputs():
    points = [{"id": "cb", "kind": "command_double", "address": 0}, {"id": "tap", "kind": "command_step", "address": 1},
              {"id": "sp", "kind": "command_setpoint", "address": 0, "format": "float"},
              {"id": "tp", "kind": "step_position", "address": 3}]
    gateway = TypedGateway("dnp3", {"bind": "127.0.0.1", "port": 20211, "outstation_address": 10, "master_address": 1},
                           points, [{"id": "tp", "value": 5, "quality": 0}])
    try:
        lines = run("studio_test_dnp3_master.exe", "127.0.0.1", 20211, 1, 10, "scan", "crob:0:close", "crob:0:trip:direct",
                    "crob:1:pulse_on", "crob:1:off:direct", "ao:0:12.5", "ao:0:7:v2:direct", "ao:0:70000:v1:direct",
                    "ao:0:999:direct", "ao:5:1:direct")
        statuses = [item["status"] for item in lines if "command" in item]
        assert statuses == ["SUCCESS"] * 7 + ["HARDWARE_ERROR", "NOT_SUPPORTED"]
        values = {(item["kind"], item["index"]): item for item in lines if "kind" in item}
        assert values[("analog", 3)]["value"] == 5
        handed = [(point, value, wire["function"], wire.get("code", wire.get("variation"))) for point, value, wire in
                  gateway.commands]
        assert handed == [
            ("cb", True, "select", "close"), ("cb", True, "operate", "close"), ("cb", False, "direct", "trip"),
            ("tap", True, "select", "pulse_on"), ("tap", True, "operate", "pulse_on"), ("tap", False, "direct", "latch_off"),
            ("sp", 12.5, "select", 3), ("sp", 12.5, "operate", 3), ("sp", 7, "direct", 2), ("sp", 70000, "direct", 1),
            ("sp", 999, "direct", 3)]
    finally:
        gateway.close()


def test_modbus_set_point_registers_and_tap_position():
    points = [{"id": "cb", "kind": "command_single", "address": 0},
              {"id": "s16", "kind": "command_setpoint", "address": 100, "format": "int16"},
              {"id": "sfl", "kind": "command_setpoint", "address": 102, "format": "float", "word_order": "high_first"},
              {"id": "s32", "kind": "command_setpoint", "address": 104, "format": "int32", "word_order": "low_first"},
              {"id": "tp", "kind": "step_position", "address": 110}]
    gateway = TypedGateway("modbus", {"bind": "127.0.0.1", "port": 15030, "unit_id": 1}, points,
                           [{"id": "tp", "value": -3, "quality": 0}])
    try:
        with socket.create_connection(("127.0.0.1", 15030), timeout=5) as sock:
            assert modbus(sock, 1, bytes([6, 0, 100, 0xFF, 0xF6])) == bytes([6, 0, 100, 0xFF, 0xF6])  # int16 -10
            float_words = struct.pack(">f", 12.5)
            assert modbus(sock, 1, bytes([16, 0, 102, 0, 2, 4]) + float_words) == bytes([16, 0, 102, 0, 2])
            int32 = struct.pack(">I", 70000)
            low_first = int32[2:] + int32[:2]
            assert modbus(sock, 1, bytes([16, 0, 104, 0, 2, 4]) + low_first) == bytes([16, 0, 104, 0, 2])
            assert modbus(sock, 1, bytes([6, 0, 102, 0, 1])) == bytes([0x86, 3])  # 32-bit needs FC16
            assert modbus(sock, 1, bytes([6, 0, 120, 0, 1])) == bytes([0x86, 2])  # no set point there
            assert modbus(sock, 1, bytes([6, 0, 100, 0x03, 0xE7])) == bytes([0x86, 4])  # 999 refused by the engine
            held = modbus(sock, 1, bytes([3, 0, 100, 0, 6]))
            # registers 100 (int16), 101 (unused), 102-103 (float), 104-105 (int32, low word first)
            assert held[2:4] == bytes([0xFF, 0xF6]) and held[6:10] == float_words and held[10:14] == low_first
            assert modbus(sock, 1, bytes([4, 0, 110, 0, 1])) == bytes([4, 2, 0xFF, 0xFD])  # tap position -3
            assert modbus(sock, 1, bytes([5, 0, 0, 0xFF, 0])) == bytes([5, 0, 0, 0xFF, 0])  # coils unchanged
        assert [(point, value) for point, value, _ in gateway.commands] == [
            ("s16", -10), ("sfl", 12.5), ("s32", 70000), ("s16", 999), ("cb", True)]
        assert [wire for _, _, wire in gateway.commands][:2] == [{"function": 6}, {"function": 16}]
    finally:
        gateway.close()
