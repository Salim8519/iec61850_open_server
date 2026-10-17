"""End to end: an independent IEC 104 master commands a simulated breaker through the
gateway, the IED operates over MMS, and the new position comes back spontaneously. The
gateway starts in autonomous mode: smart point list and Run start by themselves."""
import json
import os
import subprocess
from pathlib import Path

import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_gateway.exe").exists(), reason="Gateway build required")


def master(*steps, port=12406):
    done = subprocess.run([str(BIN / "studio_test_iec104_master.exe"), "127.0.0.1", str(port), "1", *steps],
                          capture_output=True, text=True, timeout=60)
    return [json.loads(line) for line in done.stdout.splitlines() if line.startswith("{")]


def test_iec104_master_operates_a_simulated_breaker(tmp_path):
    app = create_app(Settings(tmp_path, BIN / "studio_ied.exe", tmp_path), "gateway-token")
    app.extensions["startup"].wait()
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "gateway-token"})
    ied = app.extensions["projects"].add_template("GwBay")[0]
    ied.update(ip="127.50.0.225", port=18115)
    runtime = app.extensions["runtime"]
    try:
        config = client.put("/api/v1/tools/gateway/config", json={"protocol": "iec104", "listen": "this_pc",
                                                                   "settings": {"port": 12406}})
        assert config.status_code == 200, config.json
        # No point list yet: Start proposes the points. Run is separate and stays off until pressed.
        started = client.post("/api/v1/tools/gateway/start").json
        assert started["state"] == "running" and started["run_state"] == "stopped" and started["auto_start"] is True
        activity = client.get("/api/v1/activity").json
        assert activity["gateway"]["state"] == "running" and activity["gateway"]["port"] == 12406
        runtime.start()
        proposal = client.get("/api/v1/tools/gateway/points?protocol=iec104").json["points"]
        status_ioa = next(p["address"] for p in proposal if p["ref"] == "GwBayCTRL/XCBR1.Pos.stVal")
        command_ioa = next(p["address"] for p in proposal if p["ref"] == "GwBayCTRL/XCBR1.Pos")
        assert not any(p["ref"].endswith(".Mod.stVal") or ".NamPlt." in p["ref"] for p in proposal)  # smart list
        before = {o["ioa"]: o for frame in master("gi") if frame["cot"] == 20 for o in frame["objects"]}
        assert before[status_ioa]["quality"] == 0
        target = 1 if before[status_ioa]["value"] == 2 else 2
        frames = master(f"dc:{command_ioa}:{target}:select", f"dc:{command_ioa}:{target}", "wait:1500")
        answers = [(f["cot"], f["negative"]) for f in frames if f["type"] == 46]
        assert answers == [(7, False), (7, False), (10, False)]
        spont = [o for f in frames if f["cot"] == 3 for o in f["objects"] if o["ioa"] == status_ioa]
        assert spont and spont[-1]["value"] == target
        log = client.get("/api/v1/tools/gateway/commands").json["entries"]
        assert log[-1]["applied"].startswith("operated") and "is now" in log[-1]["observed"]
        point_id = next(p["id"] for p in proposal if p["ref"] == "GwBayCTRL/XCBR1.Pos")
        unconfirmed = client.post("/api/v1/tools/gateway/command", json={"point_id": point_id, "value": target != 2})
        assert unconfirmed.status_code == 400 and "Confirm" in unconfirmed.json["error"]
        manual = client.post("/api/v1/tools/gateway/command",
                             json={"point_id": point_id, "value": target != 2, "confirm": True}).json
        assert manual == {"ok": True, "reason": ""}
        entry = client.get("/api/v1/tools/gateway/commands").json["entries"][-1]
        assert entry["protocol"] == "manual" and entry["applied"].startswith("operated")
        traffic = client.get("/api/v1/tools/gateway/traffic").json["entries"]
        assert any("select confirmed" in entry["text"] for entry in traffic)
        table = client.get("/api/v1/tools/gateway/points?protocol=iec104").json["points"]
        assert next(row for row in table if row["address"] == status_ioa)["online"] is True
        assert client.post("/api/v1/tools/gateway/stop").json["auto_start"] is False  # user stop is remembered
    finally:
        app.extensions["gateway"].stop()
        runtime.close()
