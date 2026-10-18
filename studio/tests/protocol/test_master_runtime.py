"""End to end through the API: ScadaBench as SCADA master (104 controlling station) discovers
every point of a device by general interrogation, names and unticks points, sends a
confirmed command and remembers the connection. The device is studio_gateway."""
import time

import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings
from test_gateway_wire import BIN, Gateway

pytestmark = pytest.mark.skipif(not (BIN / "studio_master.exe").exists(), reason="Master build required")


def wait_for(check, timeout=10):
    end = time.time() + timeout
    while time.time() < end:
        result = check()
        if result:
            return result
        time.sleep(0.1)
    return check()


def test_master_role_discovers_names_and_commands_a_device(tmp_path):
    device = Gateway("iec104", {"bind": "127.0.0.1", "port": 12411, "common_address": 1})
    app = create_app(Settings(tmp_path, BIN / "studio_ied.exe", tmp_path), "master-token")
    app.extensions["startup"].wait()
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "master-token"})
    try:
        refused = client.post("/api/v1/tools/master/connect")
        assert refused.status_code == 400 and "SCADA master first" in refused.json["error"]
        assert client.post("/api/v1/tools/gateway/role", json={"role": "master"}).json == {"role": "master"}
        gateway_start = client.post("/api/v1/tools/gateway/start")
        assert gateway_start.status_code == 400 and "Station gateway first" in gateway_start.json["error"]
        bad = client.put("/api/v1/tools/master/config", json={"protocol": "iec104", "host": "bad host!"})
        assert bad.status_code == 400 and "device address" in bad.json["error"]
        saved = client.put("/api/v1/tools/master/config", json={"protocol": "iec104", "host": "127.0.0.1",
                                                                "settings": {"port": 12411}}).json
        assert saved["host"] == "127.0.0.1" and saved["settings"]["iec104"]["gi_minutes"] == 15
        status = client.post("/api/v1/tools/master/connect").json
        assert status["state"] == "connected" and status["auto_connect"] is True
        points = wait_for(lambda: len(client.get("/api/v1/tools/master/points").json["points"]) >= 4)
        rows = {row["key"]: row for row in client.get("/api/v1/tools/master/points").json["points"]}
        assert points and rows["double:2001"]["value_text"] == "closed (ON)" and rows["double:2001"]["ticked"]
        assert rows["integer:4001" if "integer:4001" in rows else "scaled:4001"]["quality_text"] == "invalid"
        assert rows["double:2001"]["command"] == "double" and rows["float:3001"]["command"] == "setpoint_float"
        client.put("/api/v1/tools/master/label", json={"key": "double:2001", "label": "Feeder 1 breaker"})
        client.put("/api/v1/tools/master/selection", json={"unticked": ["single:1001"]})
        rows = {row["key"]: row for row in client.get("/api/v1/tools/master/points").json["points"]}
        assert rows["double:2001"]["label"] == "Feeder 1 breaker" and rows["single:1001"]["ticked"] is False
        csv_text = client.get("/api/v1/tools/master/points.csv").get_data(as_text=True)
        assert "Feeder 1 breaker" in csv_text and csv_text.startswith("group,address,label")
        unconfirmed = client.post("/api/v1/tools/master/command", json={"kind": "double", "address": 6001, "value": 2})
        assert unconfirmed.status_code == 400 and "Confirm" in unconfirmed.json["error"]
        done = client.post("/api/v1/tools/master/command", json={"kind": "double", "address": 6001, "value": 2,
                                                                 "select": True, "confirm": True}).json
        assert done["ok"] and done["terminated"] and device.commands == [("cb", True)]
        refused = client.post("/api/v1/tools/master/command", json={"kind": "single", "address": 6002, "value": 1,
                                                                    "confirm": True}).json
        assert refused["ok"] is False and "refused" in refused["text"]
        log = client.get("/api/v1/tools/master/commands").json["entries"]
        assert [entry["state"] for entry in log] == ["confirmed", "refused"]
        traffic = client.get("/api/v1/tools/master/traffic").json["entries"]
        assert any("All values received" in entry["text"] for entry in traffic)
        activity = client.get("/api/v1/activity").json["master"]
        assert activity["state"] == "connected" and activity["points"] >= 4
        assert client.post("/api/v1/tools/master/disconnect").json["auto_connect"] is False
        client.put("/api/v1/tools/master/config", json={"protocol": "dnp3", "host": "127.0.0.1"})
        wrong = client.post("/api/v1/tools/master/command", json={"kind": "double", "address": 1, "value": 2, "confirm": True})
        assert wrong.status_code == 400 and "not available for dnp3" in wrong.json["error"]
    finally:
        app.extensions["master"].disconnect()
        app.extensions["runtime"].close()
        device.close()
