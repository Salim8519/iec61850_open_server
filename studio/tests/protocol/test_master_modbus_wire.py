"""ScadaBench as Modbus TCP client (studio_master) against our Modbus server (studio_gateway)
and an independent stand-in server written here. Spec: docs/specs/scada-master.md."""
import socket
import struct
import threading
import time

import pytest
from test_gateway_wire import POINTS, Gateway
from test_master_wire import BIN, Master

pytestmark = pytest.mark.skipif(not (BIN / "studio_master.exe").exists(), reason="Master build required")


class StandIn:
    """A minimal independent Modbus server: holding registers 100..109, coils 0..3; others refused."""

    def __init__(self, port):
        self.registers = {100 + index: index * 11 for index in range(10)}
        self.coils = {index: False for index in range(4)}
        self.server = socket.create_server(("127.0.0.1", port))
        self.thread = threading.Thread(target=self.serve, daemon=True)
        self.thread.start()

    def serve(self):
        try:
            connection, _ = self.server.accept()
        except OSError:
            return
        with connection:
            while True:
                header = connection.recv(7)
                if len(header) < 7:
                    return
                transaction, _, length, unit = struct.unpack(">HHHB", header)
                pdu = connection.recv(length - 1)
                answer = self.answer(pdu)
                connection.sendall(struct.pack(">HHHB", transaction, 0, len(answer) + 1, unit) + answer)

    def answer(self, pdu):
        function, address, value = pdu[0], *struct.unpack(">HH", pdu[1:5])
        if function == 3 and all(item in self.registers for item in range(address, address + value)):
            words = b"".join(struct.pack(">H", self.registers[item]) for item in range(address, address + value))
            return bytes([3, len(words)]) + words
        if function == 5 and address in self.coils:
            self.coils[address] = value == 0xFF00
            return pdu[:5]
        return bytes([function | 0x80, 2 if function in (3, 5) else 1])

    def close(self):
        self.server.close()


def test_modbus_client_reads_ranges_from_our_gateway_and_writes_a_coil():
    points = [dict(POINTS[0], address=0), dict(POINTS[1], address=5), {"id": "mv", "kind": "float", "address": 10},
              {"id": "iv", "kind": "integer", "address": 12}, {"id": "cb", "kind": "command_double", "address": 0}]
    gateway = Gateway("modbus", {"bind": "127.0.0.1", "port": 15031, "unit_id": 3}, points)
    master = Master()
    try:
        master.process.request("connect", protocol="modbus", settings={"host": "127.0.0.1", "port": 15031, "unit_id": 3})
        answer = master.process.request("poll", ranges=[
            {"table": "discrete_inputs", "start": 0, "count": 1, "data_type": "bool"},
            {"table": "holding_registers", "start": 10, "count": 1, "data_type": "float32"},
            {"table": "holding_registers", "start": 12, "count": 1, "data_type": "int32"},
            {"table": "input_registers", "start": 5, "count": 1, "data_type": "uint16"}])
        assert answer == {"values": 4, "failed": []}
        found = {(value["group"], value["address"]): value for value in master.values}
        assert found[("discrete_input", 0)]["value"] == 1
        assert found[("holding_register", 10)]["value"] == pytest.approx(230.5)
        assert found[("holding_register", 12)]["value"] == 42 and found[("input_register", 5)]["value"] == 2
        done = master.process.request("command", kind="coil", address=0, value=1)
        assert done["ok"] and gateway.commands == [("cb", True)]
        refused = master.process.request("command", kind="register", address=0, value=1)
        assert not refused["ok"] and "illegal function" in refused["steps"][0]["text"]
    finally:
        master.close()
        gateway.close()


def test_modbus_client_finds_registers_on_an_independent_server_and_explains_refusals():
    server = StandIn(15032)
    master = Master()
    try:
        master.process.request("connect", protocol="modbus", settings={"host": "127.0.0.1", "port": 15032})
        found = master.process.request("probe", timeout=60, table="holding_registers", start=90, end=129)
        assert found["found"] == [{"table": "holding_registers", "start": 100, "count": 10}]
        assert found["requests"] <= 40  # never more requests than one per address
        answer = master.process.request("poll", ranges=[
            {"table": "holding_registers", "start": 100, "count": 10, "data_type": "uint16"},
            {"table": "holding_registers", "start": 200, "count": 2, "data_type": "uint16"}])
        assert answer["failed"] == [{"range": 2, "text": "illegal data address (no such address)"}]
        values = {value["address"]: value for value in master.values}
        assert values[109]["value"] == 99 and values[200]["quality"] == 1 and values[200]["value"] is None
        assert master.process.request("command", kind="coil", address=2, value=1)["ok"] and server.coils[2]
        with pytest.raises(RuntimeError, match="1 to 10000 addresses"):
            master.process.request("probe", table="holding_registers", start=0, end=20000)
        with pytest.raises(RuntimeError, match="no interrogation"):
            master.process.request("interrogate")
    finally:
        master.close()
        server.close()


def test_find_registers_stops_on_request_and_commands_still_go_through():
    # 0178 (C14): Find registers runs on its own thread; probe_stop ends it after the request on the wire.
    server = StandIn(15033)
    master = Master()
    try:
        master.process.request("connect", protocol="modbus", settings={"host": "127.0.0.1", "port": 15033})
        result = {}
        worker = threading.Thread(target=lambda: result.update(master.process.request(
            "probe", timeout=60, table="holding_registers", start=0, end=9999)))
        worker.start()
        time.sleep(1.5)
        assert master.process.request("command", kind="coil", address=1, value=1)["ok"] and server.coils[1]
        started = time.time()
        assert master.process.request("probe_stop") is True
        worker.join(10)
        assert not worker.is_alive() and time.time() - started < 2
        assert result["stopped"] is True and 5 < result["requests"] < 200
        assert master.process.request("probe_stop") is True  # nothing running: nothing happens
    finally:
        master.close()
        server.close()
