"""End to end through the API: ScadaBench as Modbus client finds the registers of an
independent server, reads them every poll interval and writes a coil after confirmation."""
import time

import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings
from test_master_modbus_wire import StandIn
from test_master_wire import BIN

pytestmark = pytest.mark.skipif(not (BIN / "studio_master.exe").exists(), reason="Master build required")


def wait_for(check, timeout=15):
    end = time.time() + timeout
    while time.time() < end and not check():
        time.sleep(0.1)
    return check()


def test_modbus_client_finds_reads_and_writes_through_the_api(tmp_path):
    server = StandIn(15041)
    app = create_app(Settings(tmp_path, BIN / "studio_ied.exe", tmp_path), "modbus-token")
    app.extensions["startup"].wait()
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "modbus-token"})
    try:
        client.post("/api/v1/tools/gateway/role", json={"role": "master"})
        client.put("/api/v1/tools/master/config", json={"protocol": "modbus", "host": "127.0.0.1",
                                                        "settings": {"port": 15041, "poll_ms": 300}})
        status = client.post("/api/v1/tools/master/connect").json
        assert status["state"] == "connected" and "Add the Modbus ranges" in status["message"]
        too_wide = client.post("/api/v1/tools/master/probe", json={"table": "holding_registers", "start": 0, "end": 10000})
        assert too_wide.status_code == 400 and "10000 addresses" in too_wide.json["error"]
        started = client.post("/api/v1/tools/master/probe", json={"table": "holding_registers", "start": 96, "end": 111}).json
        assert started["state"] == "running"
        assert wait_for(lambda: client.get("/api/v1/tools/master").json["probe"]["state"] == "done")
        ranges = client.post("/api/v1/tools/master/probe/ranges", json={"data_type": "uint16"}).json["modbus_ranges"]
        assert ranges == [{"table": "holding_registers", "start": 100, "count": 10, "data_type": "uint16", "word_order": "big"}]
        assert wait_for(lambda: len(client.get("/api/v1/tools/master/points").json["points"]) == 10)
        rows = {row["address"]: row for row in client.get("/api/v1/tools/master/points").json["points"]}
        assert rows[109]["value_text"] == "99" and rows[109]["command"] == "register" and rows[109]["cause"] == "polled"
        server.registers[109] = 7
        assert wait_for(lambda: {row["address"]: row for row in client.get("/api/v1/tools/master/points").json["points"]}
                        [109]["value"] == 7)
        done = client.post("/api/v1/tools/master/command", json={"kind": "coil", "address": 3, "value": 1,
                                                                 "confirm": True}).json
        assert done["ok"] and server.coils[3]
        assert client.get("/api/v1/activity").json["master"]["points"] == 10
    finally:
        app.extensions["master"].disconnect()
        app.extensions["runtime"].close()
        server.close()
