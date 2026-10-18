"""Node-aware operations update the explicitly mapped state over native MMS."""
import json
import os
import subprocess
import time
from pathlib import Path
import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings
from studio_app.scl.templates import make_template

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_ied.exe").exists(),
                                reason="Native build required")


def probe(ip, port, reference):
    result = subprocess.run([str(BIN / "studio_probe.exe"), ip, str(port), "read",
                             reference, "ST"], capture_output=True, text=True, timeout=15)
    assert result.returncode == 0, result.stdout + result.stderr
    return json.loads(result.stdout)["value"]


def control(ip, port, reference, value):
    result = subprocess.run([str(BIN / "studio_probe.exe"), ip, str(port), "control",
                             reference, "true" if value else "false"],
                            capture_output=True, text=True, timeout=15)
    assert result.returncode == 0, result.stdout + result.stderr


def await_value(ip, port, reference, expected):
    for _ in range(40):
        if probe(ip, port, reference) in ({"true", "1"} if expected else {"false", "0"}):
            return
        time.sleep(0.05)
    raise AssertionError(f"{reference} did not become {expected}")


def test_api_operation_drives_separately_selected_double_point(tmp_path):
    app = create_app(Settings(tmp_path, BIN / "studio_ied.exe", tmp_path), "control-token")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "control-token"})
    projects = app.extensions["projects"]
    ied = projects.add_template("Mapped")[0]
    ied.update(ip="127.50.0.223", port=18113)
    status = "MappedCTRL/XSWI1.Pos.stVal"
    projects.diagram({"nodes": [{"id": "switch", "position": {"x": 0, "y": 0},
                      "data": {"label": "Switch", "kind": "CB",
                               "state_mapping": {"mode": "direct", "source": {
                                   "ied_id": ied["id"], "ref": status, "report_ref": None}},
                               "control_binding": {"ied_id": ied["id"],
                                                   "ref": "MappedCTRL/XCBR1.Pos"}}}],
                      "edges": []})
    runtime = app.extensions["runtime"]
    try:
        runtime.start()
        closed = client.post("/api/v1/equipment/switch/operate", json={"value": True})
        assert closed.status_code == 200
        assert closed.json["feedback"] == [{"ied_id": ied["id"], "ref": status, "value": 2}]
        assert runtime.supervisor.read(ied["id"], status) == 2
        wire_closed = probe(ied["ip"], ied["port"], status)
        opened = client.post("/api/v1/equipment/switch/operate", json={"value": False})
        assert opened.status_code == 200
        assert runtime.supervisor.read(ied["id"], status) == 1
        assert probe(ied["ip"], ied["port"], status) != wire_closed
        states = [event["value"] for event in app.extensions["events"].after(0)
                  if event["kind"] == "equipment_state" and event["node_id"] == "switch"]
        assert 2 in states and states[-1] == 1
    finally:
        runtime.close()


def test_concen_originator_gets_standard_enhanced_sbo_close(tmp_path):
    """Regression for 2026-10-18: a GE concentrator (orIdent CONCEN) Close must close on Oper."""
    app = create_app(Settings(tmp_path, BIN / "studio_ied.exe", tmp_path), "standard-token")
    projects = app.extensions["projects"]
    source = make_template("MappedStd").replace(b"<Val>3</Val>", b"<Val>4</Val>")
    ied = projects.add_scl(source, "standard.cid")[0]
    ied.update(ip="127.50.0.227", port=18117)
    control_ref = "MappedStdCTRL/XCBR1.Pos"
    status_ref = control_ref + ".stVal"
    projects.diagram({"nodes": [{"id": "breaker", "position": {"x": 0, "y": 0},
                      "data": {"label": "Breaker", "kind": "CB",
                               "state_mapping": {"mode": "direct", "source": {
                                   "ied_id": ied["id"], "ref": status_ref, "report_ref": None}},
                               "control_binding": {"ied_id": ied["id"], "ref": control_ref}}}],
                      "edges": []})
    runtime = app.extensions["runtime"]
    try:
        runtime.start()
        runtime.update(ied["id"], status_ref, 1)
        result = subprocess.run([str(BIN / "studio_probe.exe"), ied["ip"], str(ied["port"]),
                                 "sbo-sequence", control_ref, "true", "0", "CONCEN"],
                                capture_output=True, text=True, timeout=15)
        assert result.returncode == 0, result.stdout + result.stderr
        outcome = json.loads(result.stdout)
        assert outcome["ok"], outcome
        assert outcome["after_select"] == outcome["before"], "Select alone must never move the switch"
        for _ in range(40):
            observed = [event for event in app.extensions["events"].after(0)
                        if event["kind"] == "equipment_feedback_observed"]
            if observed:
                break
            time.sleep(0.05)
        assert observed[-1]["wire_value"] is True and observed[-1]["action"] == "close"
        assert runtime.supervisor.read(ied["id"], status_ref) == 2
    finally:
        runtime.close()


def test_trip_polarity_and_both_push_button_modes(tmp_path):
    app = create_app(Settings(tmp_path, BIN / "studio_ied.exe", tmp_path), "button-token")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "button-token"})
    projects = app.extensions["projects"]
    ied = projects.add_template("Buttons")[0]
    ied.update(ip="127.50.0.224", port=18114)
    boolean = lambda ref: {"mode": "direct", "source": {
        "ied_id": ied["id"], "ref": ref, "report_ref": None}}
    spc = {"ied_id": ied["id"], "ref": "ButtonsCTRL/GGIO1.SPCSO1"}
    projects.diagram({"nodes": [
        {"id": "trip", "position": {"x": 0, "y": 0}, "data": {
            "label": "Trip", "kind": "TRIP_RELAY",
            "trip_command_polarity": "operate_false", "trip_state_polarity": "operated_true",
            "state_mapping": boolean("ButtonsCTRL/XCBR1.Loc.stVal"), "control_binding": spc}},
        {"id": "trip-copy", "position": {"x": 0, "y": 1}, "data": {
            "label": "Trip copy", "kind": "TRIP_RELAY",
            "trip_command_polarity": "operate_false", "trip_state_polarity": "operated_true",
            "state_mapping": boolean("ButtonsCTRL/XSWI2.Loc.stVal"), "control_binding": spc}},
        {"id": "trip-native", "position": {"x": 0, "y": 2}, "data": {
            "label": "Trip native", "kind": "TRIP_RELAY",
            "trip_command_polarity": "operate_false", "trip_state_polarity": "operated_true",
            "state_mapping": boolean("ButtonsCTRL/GGIO1.SPCSO1.stVal"), "control_binding": spc}},
        {"id": "momentary", "position": {"x": 1, "y": 0}, "data": {
            "label": "Momentary", "kind": "PUSH_BUTTON", "push_mode": "spc_momentary",
            "push_value": False, "control_binding": spc}},
        {"id": "latching", "position": {"x": 2, "y": 0}, "data": {
            "label": "Latching", "kind": "PUSH_BUTTON", "push_mode": "latching_status",
            "state_mapping": boolean("ButtonsCTRL/XSWI1.Loc.stVal")}}], "edges": []})
    runtime = app.extensions["runtime"]
    try:
        runtime.start()
        operated = client.post("/api/v1/equipment/trip/operate", json={"action": "operate"})
        assert operated.status_code == 200 and operated.json["wire_value"] is False
        assert operated.json["feedback"][0]["value"] is True
        await_value(ied["ip"], ied["port"], spc["ref"] + ".stVal", True)
        reset = client.post("/api/v1/equipment/trip/operate", json={"action": "reset"})
        assert reset.status_code == 200 and reset.json["wire_value"] is True
        assert reset.json["feedback"][0]["value"] is False
        await_value(ied["ip"], ied["port"], spc["ref"] + ".stVal", False)
        before = len([event for event in app.extensions["events"].after(0)
                      if event["kind"] == "control_applied" and event["ref"] == spc["ref"]])
        pressed = client.post("/api/v1/equipment/momentary/operate", json={"action": "press"})
        assert pressed.status_code == 200 and pressed.json["wire_value"] is False
        after = [event for event in app.extensions["events"].after(0)
                 if event["kind"] == "control_applied" and event["ref"] == spc["ref"]]
        assert len(after) == before + 1
        assert probe(ied["ip"], ied["port"], spc["ref"] + ".stVal") in {"true", "1"}
        state_ref = "ButtonsCTRL/XCBR1.Loc.stVal"
        await_value(ied["ip"], ied["port"], state_ref, True)
        before_feedback = len([event for event in app.extensions["events"].after(0)
                               if event["kind"] == "signal" and event.get("ref") == state_ref])
        reset_again = client.post("/api/v1/equipment/trip/operate", json={"action": "reset"})
        assert reset_again.status_code == 200
        await_value(ied["ip"], ied["port"], state_ref, False)
        time.sleep(0.1)
        after_feedback = [event for event in app.extensions["events"].after(0)
                          if event["kind"] == "signal" and event.get("ref") == state_ref]
        assert len(after_feedback) == before_feedback + 1
        control(ied["ip"], ied["port"], spc["ref"], False)
        await_value(ied["ip"], ied["port"], state_ref, True)
        await_value(ied["ip"], ied["port"], "ButtonsCTRL/XSWI2.Loc.stVal", True)
        control(ied["ip"], ied["port"], spc["ref"], True)
        await_value(ied["ip"], ied["port"], state_ref, False)
        await_value(ied["ip"], ied["port"], "ButtonsCTRL/XSWI2.Loc.stVal", False)
        observed = [event for event in app.extensions["events"].after(0)
                    if event["kind"] == "equipment_feedback_observed"
                    and event["node_id"] == "trip"]
        assert [(event["action"], event["wire_value"]) for event in observed[-2:]] == [
            ("operate", False), ("reset", True)]
        before_states = len([event for event in app.extensions["events"].after(0)
                             if event["kind"] == "equipment_state"
                             and event["node_id"] == "trip"])
        control(ied["ip"], ied["port"], spc["ref"], True)
        for _ in range(40):
            trip_states = [event for event in app.extensions["events"].after(0)
                           if event["kind"] == "equipment_state"
                           and event["node_id"] == "trip"]
            if len(trip_states) > before_states:
                break
            time.sleep(0.05)
        assert len(trip_states) > before_states and trip_states[-1]["value"] is False
        live = {item["node_id"]: item["value"]
                for item in client.get("/api/v1/equipment/states").json["states"]}
        assert live["trip"] is False and live["trip-copy"] is False
        first = client.post("/api/v1/equipment/latching/operate", json={"action": "press"})
        second = client.post("/api/v1/equipment/latching/operate", json={"action": "press"})
        assert first.json["feedback"][0]["value"] is True
        assert second.json["feedback"][0]["value"] is False
        signals = [event for event in app.extensions["events"].after(0)
                   if event["kind"] == "signal" and event.get("source") == "equipment_button"]
        assert [event["value"] for event in signals[-2:]] == [True, False]
    finally:
        runtime.close()


def test_self_resetting_push_button_returns_to_rest(tmp_path):
    app = create_app(Settings(tmp_path, BIN / "studio_ied.exe", tmp_path), "reset-token")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "reset-token"})
    projects = app.extensions["projects"]
    ied = projects.add_template("Reset")[0]
    ied.update(ip="127.50.0.225", port=18115)
    projects.diagram({"nodes": [
        {"id": "reset", "position": {"x": 0, "y": 0}, "data": {
            "label": "Reset", "kind": "PUSH_BUTTON", "push_mode": "self_resetting",
            "push_rest_value": False, "push_reset_ms": 200,
            "state_mapping": {"mode": "direct", "source": {
                "ied_id": ied["id"], "ref": "ResetCTRL/XSWI1.Loc.stVal",
                "report_ref": None}}}}], "edges": []})
    runtime = app.extensions["runtime"]
    try:
        runtime.start()
        pressed = client.post("/api/v1/equipment/reset/operate", json={"action": "press"})
        assert pressed.status_code == 200 and pressed.json["action"] == "press"
        assert pressed.json["feedback"][0]["value"] is True
        await_value(ied["ip"], ied["port"], "ResetCTRL/XSWI1.Loc.stVal", True)
        await_value(ied["ip"], ied["port"], "ResetCTRL/XSWI1.Loc.stVal", False)
    finally:
        runtime.close()
