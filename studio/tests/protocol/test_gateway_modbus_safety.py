"""Modbus gateway safety (fix plan 2026-11-19, A12-A14) against studio_gateway with a raw-socket
master: overlapping registers refuse the start, Stop does not wait for an unanswered command,
and idle connections are closed so a real master can still connect."""
import socket
import struct
import time

import pytest
from studio_app.runtime.worker_process import WorkerProcess
from test_gateway_wire import BIN, modbus

pytestmark = pytest.mark.skipif(not (BIN / "studio_gateway.exe").exists(), reason="Gateway build required")
IDLE_TIMEOUT_S = 60  # modbus_server.cpp IDLE_TIMEOUT_S


def settings(port):
    return {"bind": "127.0.0.1", "port": port, "unit_id": 1}


def closed_by_gateway(sock):
    sock.settimeout(5)
    try:
        return sock.recv(1) == b""
    except ConnectionResetError:
        return True


def test_overlapping_registers_refuse_the_start():
    """A13: FC3/FC4 merge kinds and a float uses 2 registers, so float 10 and float 11 overlap."""
    process = WorkerProcess(BIN / "studio_gateway.exe", lambda event: None)
    try:
        overlapping = [{"id": "mv1", "kind": "float", "address": 10}, {"id": "mv2", "kind": "float", "address": 11}]
        with pytest.raises(RuntimeError, match="Points mv1 and mv2 both use Modbus register 11"):
            process.request("start", protocol="modbus", settings=settings(15060), points=overlapping)
        mixed = [{"id": "dp", "kind": "double", "address": 9}, {"id": "mv1", "kind": "float", "address": 10},
                 {"id": "s32", "kind": "command_setpoint", "address": 11, "format": "int32"}]
        with pytest.raises(RuntimeError, match="Points mv1 and s32 both use Modbus register 11"):
            process.request("start", protocol="modbus", settings=settings(15060), points=mixed)
        separate = [{"id": "dp", "kind": "double", "address": 9}, {"id": "mv1", "kind": "float", "address": 10},
                    {"id": "s16", "kind": "command_setpoint", "address": 12, "format": "int16"},
                    {"id": "tp", "kind": "step_position", "address": 13},
                    {"id": "sp", "kind": "single", "address": 10},
                    {"id": "cb", "kind": "command_double", "address": 10}]
        assert process.request("start", protocol="modbus", settings=settings(15060), points=separate)["port"] == 15060
    finally:
        process.close()


def test_halt_refuses_a_waiting_command_at_once():
    """A14: the fake engine never answers; halt must not wait 15 s per coil of an FC15."""
    commands = []
    process = WorkerProcess(BIN / "studio_gateway.exe",
                            lambda event: commands.append(event) if event.get("event") == "gw_command" else None)
    try:
        points = [{"id": f"c{index}", "kind": "command_single", "address": index} for index in range(3)]
        process.request("start", protocol="modbus", settings=settings(15061), points=points)
        with socket.create_connection(("127.0.0.1", 15061), timeout=20) as sock:
            sock.sendall(struct.pack(">HHHB", 1, 0, 8, 1) + bytes([15, 0, 0, 0, 3, 1, 0b111]))
            deadline = time.monotonic() + 5
            while not commands and time.monotonic() < deadline:
                time.sleep(0.05)
            assert len(commands) == 1
            began = time.monotonic()
            process.request("halt", timeout=10)
            assert time.monotonic() - began < 1.5
            assert len(commands) == 1  # the FC15 loop stopped at the first refused coil
            process.request("command_result", token=commands[0]["token"], ok=True)  # late: dropped
    finally:
        process.close()


def test_idle_connections_are_closed_so_a_master_can_connect():
    """A12: four idle connections used every slot; they are closed after the idle timeout."""
    process = WorkerProcess(BIN / "studio_gateway.exe", lambda event: None)
    idle = []
    try:
        process.request("start", protocol="modbus", settings=settings(15062),
                        points=[{"id": "dp", "kind": "double", "address": 0}])
        idle = [socket.create_connection(("127.0.0.1", 15062), timeout=5) for _ in range(4)]
        time.sleep(0.5)
        with socket.create_connection(("127.0.0.1", 15062), timeout=5) as refused:
            assert closed_by_gateway(refused)  # the connection limit is reached
        time.sleep(IDLE_TIMEOUT_S + 2)
        assert all(closed_by_gateway(sock) for sock in idle)
        with socket.create_connection(("127.0.0.1", 15062), timeout=5) as master:
            assert modbus(master, 1, bytes([4, 0, 0, 0, 1])) == bytes([4, 2, 0, 0])
    finally:
        for sock in idle:
            sock.close()
        process.close()
