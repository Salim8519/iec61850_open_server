"""studio_control operates IEC 61850 controls the way a real client does (ADR-0057)."""
import json
import os
import subprocess
from pathlib import Path

import pytest
from studio_app.runtime.worker_process import WorkerProcess
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_control.exe").exists(), reason="Native build required")
IP, PORT = "127.50.0.238", 18138


@pytest.fixture
def device(tmp_path):
    model = import_scl(make_template("CtlTest"))[0]
    config = tmp_path / "device.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    controls = [dict(item, blocked=item["ref"].endswith("SPCSO1")) for item in model["controls"]]
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    worker.request("start", cfg=str(config), ip=IP, port=PORT, controls=controls, edition=2)
    yield worker
    worker.close()


@pytest.fixture
def control():
    process = WorkerProcess(BIN / "studio_control.exe", lambda event: None)
    yield process
    process.close()


def test_breaker_closes_and_opens_with_its_own_control_model(device, control):
    for value, expected in ((True, 2), (False, 1)):
        result = control.request("operate", timeout=20, ip=IP, port=PORT, ref="CtlTestCTRL/XCBR1.Pos", value=value)
        assert result["ok"] and result["operated"], result
        assert result["model"] == "direct-with-enhanced-security" and result["terminated"] is True
        assert device.request("read", ref="CtlTestCTRL/XCBR1.Pos.stVal") == expected


def test_blocked_control_reports_the_device_reason(device, control):
    result = control.request("operate", timeout=20, ip=IP, port=PORT, ref="CtlTestCTRL/GGIO1.SPCSO1", value=True)
    assert result["ok"] is False and result["operated"] is False
    assert "refused by the device" in result["error"] and "access" in result["error"].lower()


def test_unknown_object_is_explained(device, control):
    with pytest.raises(RuntimeError, match="not a controllable object"):
        control.request("operate", timeout=20, ip=IP, port=PORT, ref="CtlTestCTRL/XCBR1.Nope", value=True)


def test_unreachable_device_is_explained(control):
    with pytest.raises(RuntimeError, match="Could not connect"):
        control.request("operate", timeout=20, ip="127.50.0.239", port=18139, ref="X/Y.Z", value=True)


def test_gateway_commands_a_real_role_ied_and_respects_the_switch(device):
    """Owner decision 2026-10-17: SCADA commands reach real IEDs unless switched off."""
    from types import SimpleNamespace
    from studio_app.gateway.commands import CommandExecutor
    from studio_app.gateway.traffic_log import NumberedLog
    from studio_app.tools.control_client import ControlClient
    model = import_scl(make_template("CtlTest"))[0]
    real = dict(model, id="r", name="Relay", role="real", state="real_live", live={"ip": IP, "port": PORT})
    point = {"id": "p", "kind": "command_double", "ied": "r", "ref": "CtlTestCTRL/XCBR1.Pos"}
    config = {"real_commands": True}
    manager = SimpleNamespace(point=lambda _: point, ied=lambda _: real, config=lambda: config,
                              commands=NumberedLog(50), emit=lambda *a, **k: None,
                              control=ControlClient(BIN / "studio_control.exe"),
                              coordinator=SimpleNamespace(index={}))
    executor = CommandExecutor(manager)
    try:
        event = {"point_id": "p", "value": True, "protocol": "iec104", "peer": "test"}
        assert executor._execute(event) == (True, "")
        assert device.request("read", ref="CtlTestCTRL/XCBR1.Pos.stVal") == 2
        config["real_commands"] = False
        ok, reason = executor._execute(dict(event, value=False))
        assert not ok and "switched off" in reason
        assert device.request("read", ref="CtlTestCTRL/XCBR1.Pos.stVal") == 2  # nothing was sent
    finally:
        manager.control.close()


def test_canvas_command_names_its_origin(tmp_path):
    """Change 0186: SLD and plain commands carry orCat 3 (remote control) and orIdent "ScadaBench"
    like typed commands. The DPC gets an ST origin, which the vendor stack fills from each Oper."""
    text = make_template("CtlTest").replace(
        b'<DA name="t" bType="Timestamp" fc="ST" /><DA name="ctlModel"',
        b'<DA name="t" bType="Timestamp" fc="ST" /><DA name="origin" bType="Struct" fc="ST" type="StudioOrigin" />'
        b'<DA name="ctlModel"')
    model = import_scl(text)[0]
    config = tmp_path / "origin.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    worker = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    control = WorkerProcess(BIN / "studio_control.exe", lambda event: None)
    try:
        worker.request("start", cfg=str(config), ip="127.50.0.240", port=18140, controls=model["controls"], edition=2)
        ref = "CtlTestCTRL/XCBR1.Pos"
        assert worker.request("read", ref=ref + ".origin.orCat") == 0
        result = control.request("canvas_operate", timeout=20, ip="127.50.0.240", port=18140, ref=ref,
                                 value=True, keep_ms=0)
        assert result["ok"], result
        assert worker.request("read", ref=ref + ".origin.orCat") == 3
        observed = subprocess.run([str(BIN / "studio_probe.exe"), "127.50.0.240", "18140", "read",
                                   ref + ".origin.orIdent", "ST"], capture_output=True, text=True, timeout=10)
        assert json.loads(observed.stdout)["value"].lower() in (b"ScadaBench".hex(), "scadabench")
    finally:
        control.close()
        worker.close()
