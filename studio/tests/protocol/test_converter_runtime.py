"""Protocol converter end to end: an independent IEC 104 master (lib60870) talks to ScadaBench
as 104 controlled station, which serves what ScadaBench reads as DNP3 master from a DNP3
outstation (studio_gateway). Values flow up, a command flows down to the outstation."""
import time

import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings
from test_gateway_runtime import master as iec104_master
from test_gateway_wire import POINTS, Gateway
from test_master_wire import BIN

pytestmark = pytest.mark.skipif(not (BIN / "studio_master.exe").exists(), reason="Master build required")


def test_dnp3_device_is_served_as_iec104_and_scada_commands_reach_it(tmp_path):
    device = Gateway("dnp3", {"bind": "127.0.0.1", "port": 20231, "outstation_address": 10, "master_address": 1},
                     [dict(point, address=index) for index, point in enumerate(POINTS)])
    app = create_app(Settings(tmp_path, BIN / "studio_ied.exe", tmp_path), "converter-token")
    app.extensions["startup"].wait()
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "converter-token"})
    try:
        assert client.post("/api/v1/tools/gateway/role", json={"role": "converter"}).json == {"role": "converter"}
        early = client.post("/api/v1/tools/gateway/start")
        assert early.status_code == 400 and "Connect to the device first" in early.json["error"]
        client.put("/api/v1/tools/master/config", json={"protocol": "dnp3", "host": "127.0.0.1",
                                                        "settings": {"port": 20231, "event_seconds": 1}})
        assert client.post("/api/v1/tools/master/connect").json["state"] == "connected"
        end = time.time() + 10
        while time.time() < end and len(client.get("/api/v1/tools/master/points").json["points"]) < 4:
            time.sleep(0.1)
        client.put("/api/v1/tools/gateway/config", json={"protocol": "iec104", "listen": "this_pc",
                                                         "settings": {"port": 12431}})
        started = client.post("/api/v1/tools/gateway/start").json
        assert started["state"] == "running" and started["role"] == "converter"
        table = {row["remote"] if "remote" in row else row["id"]: row
                 for row in client.get("/api/v1/tools/gateway/points?protocol=iec104").json["points"]}
        analog = table["analog:2"]["address"]
        command = next(row["address"] for row in table.values() if row["id"] == "remote-command:binary_output:4")
        time.sleep(0.5)  # one sync cycle
        frames = iec104_master("gi", f"sc:{command}:1", "wait:1500", port=12431)
        values = {item["ioa"]: item for frame in frames if frame["cot"] == 20 for item in frame["objects"]}
        assert values[analog]["value"] == pytest.approx(230.5) and values[analog]["quality"] == 0
        assert [(f["cot"], f["negative"]) for f in frames if f["type"] == 45] == [(7, False), (10, False)]
        assert device.commands == [("cb", True)]
        log = client.get("/api/v1/tools/master/commands").json["entries"]
        assert log[-1]["who"] == "SCADA via the converter" and log[-1]["state"] == "confirmed"
        assert client.get("/api/v1/tools/gateway/commands").json["entries"][-1]["applied"].startswith("operated")
    finally:
        app.extensions["gateway"].stop()
        app.extensions["master"].disconnect()
        app.extensions["runtime"].close()
        device.close()
