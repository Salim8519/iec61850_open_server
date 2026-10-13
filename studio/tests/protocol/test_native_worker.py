import json
import os
import subprocess
import time
from pathlib import Path
import pytest
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template
from studio_app.runtime.worker_process import WorkerProcess

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_ied.exe").exists(), reason="Native build required")


def probe(*args):
    result = subprocess.run([str(BIN / "studio_probe.exe"), *args], capture_output=True,
                            text=True, timeout=15)
    assert result.returncode == 0, result.stdout + result.stderr
    return json.loads(result.stdout)


def test_real_mms_dpc_and_latching_spc_controls(tmp_path):
    model = import_scl(make_template("Native"))[0]
    config = tmp_path / "ied.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    dpc = "NativeCTRL/XCBR1.Pos"
    dpc_status = dpc + ".stVal"
    spc = "NativeCTRL/GGIO1.SPCSO1"
    spc_status = spc + ".stVal"
    events = []
    worker = WorkerProcess(BIN / "studio_ied.exe", events.append)
    try:
        worker.request("start", cfg=str(config), ip="127.50.0.222", port=18102,
                       controls=model["controls"])
        assert probe("127.50.0.222", "18102", "control", dpc, "false")["ok"]
        opened = probe("127.50.0.222", "18102", "read", dpc_status, "ST")
        assert probe("127.50.0.222", "18102", "control", dpc, "true")["ok"]
        closed = probe("127.50.0.222", "18102", "read", dpc_status, "ST")
        assert opened["value"] != closed["value"]
        assert probe("127.50.0.222", "18102", "control", spc, "true")["ok"]
        time.sleep(0.2)
        operated = probe("127.50.0.222", "18102", "read", spc_status, "ST")
        assert operated["value"] in {"true", "1"}
        assert probe("127.50.0.222", "18102", "control", spc, "false")["ok"]
        reset = probe("127.50.0.222", "18102", "read", spc_status, "ST")
        assert reset["value"] in {"false", "0"}
        applied = [event for event in events if event["event"] == "control_applied"]
        assert [(event["cdc"], event["action"], event["ctl_value"], event["value"])
                for event in applied] == [
            ("DPC", "open", False, 1), ("DPC", "close", True, 2),
            ("SPC", "set_true", True, True), ("SPC", "set_false", False, False)]
    finally:
        worker.close()


def test_enhanced_sbo_requires_matching_oper_and_reports_selection_lifecycle(tmp_path):
    source = make_template("SboBreaker").replace(b"<Val>3</Val>", b"<Val>4</Val>")
    source = source.replace(b"<Val>10000</Val>", b"<Val>200</Val>")
    model = import_scl(source)[0]
    config = tmp_path / "sbo.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    control = "SboBreakerCTRL/XCBR1.Pos"
    status = control + ".stVal"
    events = []
    worker = WorkerProcess(BIN / "studio_ied.exe", events.append)
    try:
        started = worker.request("start", cfg=str(config), ip="127.50.0.225", port=18115,
                                 controls=model["controls"], edition=2)
        assert "control_selection_events" in started["capabilities"]
        assert next(item for item in model["controls"] if item["ref"] == control)["sbo_timeout_ms"] == 200
        worker.request("update", ref=status, value=2)
        sequence = probe("127.50.0.225", "18115", "sbo-sequence", control, "false")
        assert sequence["selected"] and not sequence["repeated"]
        assert sequence["repeated_add_cause"] == 19
        assert sequence["before"] == sequence["after_select"]
        assert sequence["operated"] and sequence["after_operate"] != sequence["after_select"]
        assert probe("127.50.0.225", "18115", "sbo-cancel", control, "false")["canceled"]
        assert probe("127.50.0.225", "18115", "sbo-fail", control, "false")["ok"]
        assert probe("127.50.0.225", "18115", "sbo-disconnect", control, "false")["ok"]
        assert probe("127.50.0.225", "18115", "sbo-timeout", control, "false", "400")["ok"]
        for _ in range(40):
            states = [event["state"] for event in events
                      if event["event"] == "control_selection" and event["ref"] == control]
            if {"operated", "canceled", "failed", "disconnected", "timed_out"} <= set(states):
                break
            time.sleep(0.05)
        assert {"selected", "operated", "canceled", "failed", "disconnected", "timed_out"} <= set(states)
        assert all(event["sbo_timeout_ms"] == 200 for event in events
                   if event["event"] == "control_selection" and event["ref"] == control)
    finally:
        worker.close()


def test_explicit_concen_profile_inverts_and_executes_sbow_only(tmp_path):
    source = make_template("CompatBreaker").replace(b"<Val>3</Val>", b"<Val>4</Val>")
    model = import_scl(source)[0]
    config = tmp_path / "compat.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    control = "CompatBreakerCTRL/XCBR1.Pos"
    status = control + ".stVal"
    controls = [{**item, "compatibility": "concen_inverted_sbow"}
                if item["ref"] == control else item for item in model["controls"]]
    events = []
    worker = WorkerProcess(BIN / "studio_ied.exe", events.append)
    try:
        started = worker.request("start", cfg=str(config), ip="127.50.0.226", port=18116,
                                 controls=controls, edition=2)
        assert "concen_sbow_compatibility" in started["capabilities"]
        worker.request("update", ref=status, value=1)
        adapted = probe("127.50.0.226", "18116", "sbo-compat", control, "false")
        assert adapted["selected"] and adapted["before"] != adapted["after_select"]
        for _ in range(40):
            if any(event["event"] == "control_compatibility_applied" for event in events):
                break
            time.sleep(0.05)
        applied = [event for event in events if event["event"] == "control_applied"]
        assert len(applied) == 1
        assert applied[0]["raw_ctl_value"] is False
        assert applied[0]["ctl_value"] is True and applied[0]["action"] == "close"
        assert any(event["event"] == "control_compatibility_applied" for event in events)
        worker.request("update", ref=status, value=2)
        assert probe("127.50.0.226", "18116", "control", control, "false")["ok"]
        assert probe("127.50.0.226", "18116", "read", status, "ST")["value"] != adapted["after_select"]
    finally:
        worker.close()


def test_concen_compat_noop_keeps_native_state_and_timestamp_unchanged(tmp_path):
    source = make_template("CompatBreaker").replace(b"<Val>3</Val>", b"<Val>4</Val>")
    model = import_scl(source)[0]
    config = tmp_path / "noop.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    control = "CompatBreakerCTRL/XCBR1.Pos"
    status = control + ".stVal"
    controls = [{**item, "compatibility": "concen_inverted_sbow"}
                if item["ref"] == control else item for item in model["controls"]]
    events = []
    worker = WorkerProcess(BIN / "studio_ied.exe", events.append)
    try:
        worker.request("start", cfg=str(config), ip="127.50.0.227", port=18117,
                       controls=controls, edition=2)
        worker.request("update", ref=status, value=2)
        time.sleep(0.3)
        result = probe("127.50.0.227", "18117", "sbo-compat-noop", control, "false")
        assert result["selected"]
        assert result["before"] == result["after_select"] == "10"
        assert result["pre_t"] == result["before_t"] == result["after_select_t"]
        applied = [event for event in events if event["event"] == "control_applied"]
        assert len(applied) == 1 and applied[0]["changed"] is False
        assert applied[0]["action"] == "close" and applied[0]["value"] == 2
        assert applied[0]["raw_ctl_value"] is False and applied[0]["ctl_value"] is True
    finally:
        worker.close()
