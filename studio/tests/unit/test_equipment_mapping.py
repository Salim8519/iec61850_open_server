"""State construction and command bindings are typed and independent."""
import io
import json
from pathlib import Path
import zipfile
import pytest
from studio_app.contracts.errors import StudioError
from studio_app.projects.service import ProjectService
from studio_app.projects.mapping_migration import migrate_diagram
from studio_app.scl.templates import make_template


def source(ied, reference, report=None):
    return {"ied_id": ied["id"], "ref": reference, "report_ref": report}


def node(kind, state=None, control=None, key="equipment", **extra):
    return {"id": key, "position": {"x": 0, "y": 0},
            "data": {"kind": kind, "label": key, "state_mapping": state,
                     "control_binding": control, **extra}}


def diagram(*nodes):
    return {"nodes": list(nodes), "edges": []}


def test_switch_state_modes_and_command_only_are_accepted(tmp_path):
    projects = ProjectService(tmp_path)
    first = projects.add_template("First")[0]
    second = projects.add_template("Second")[0]
    direct = {"mode": "direct", "source": source(first, "FirstCTRL/XCBR1.Pos.stVal")}
    single = {"mode": "single_to_dbpos", "source": source(first, "FirstCTRL/XCBR1.Loc.stVal")}
    contacts = {"mode": "contacts_to_dbpos",
                "open": source(first, "FirstCTRL/XCBR1.Loc.stVal"),
                "closed": source(second, "SecondCTRL/XSWI1.Loc.stVal")}
    dpc = {"ied_id": second["id"], "ref": "SecondCTRL/XSWI2.Pos"}
    projects.diagram(diagram(node("CB", direct, dpc, "direct"),
                             node("DS", single, key="single"),
                             node("ES", contacts, key="contacts"),
                             node("CB", control=dpc, key="command-only")))
    saved = projects.project["diagram"]["nodes"]
    assert saved[2]["data"]["state_mapping"]["closed"]["ied_id"] == second["id"]
    assert saved[3]["data"]["state_mapping"] is None


def test_switch_and_trip_relay_reject_wrong_wire_types(tmp_path):
    projects = ProjectService(tmp_path)
    ied = projects.add_template("Typed")[0]
    boolean = {"mode": "direct", "source": source(ied, "TypedCTRL/XCBR1.Loc.stVal")}
    with pytest.raises(StudioError, match="Dbpos"):
        projects.diagram(diagram(node("CB", boolean)))
    duplicate = {"mode": "contacts_to_dbpos", "open": boolean["source"],
                 "closed": boolean["source"]}
    with pytest.raises(StudioError, match="different"):
        projects.diagram(diagram(node("CB", duplicate)))
    spc = {"ied_id": ied["id"], "ref": "TypedCTRL/GGIO1.SPCSO1"}
    dpc = {"ied_id": ied["id"], "ref": "TypedCTRL/XCBR1.Pos"}
    with pytest.raises(StudioError, match="DPC"):
        projects.diagram(diagram(node("CB", control=spc)))
    with pytest.raises(StudioError, match="SPC"):
        projects.diagram(diagram(node("TRIP_RELAY", boolean, dpc)))


def test_command_bindings_are_standard_iec61850_only(tmp_path):
    projects = ProjectService(tmp_path)
    ied = projects.add_template("Profile")[0]
    binding = {"ied_id": ied["id"], "ref": "ProfileCTRL/XCBR1.Pos"}
    projects.diagram(diagram(node("CB", control=binding, key="first"),
                             node("CB", control=dict(binding), key="second")))
    assert [item["data"]["control_binding"] for item in projects.project["diagram"]["nodes"]] == [binding] * 2
    with pytest.raises(Exception):
        projects.diagram(diagram(node("CB", control={**binding, "compatibility": "concen_inverted_sbow"})))


def test_trip_relay_uses_one_boolean_state_and_one_spc_command(tmp_path):
    projects = ProjectService(tmp_path)
    ied = projects.add_template("Trip")[0]
    state = {"mode": "direct", "source": source(ied, "TripCTRL/GGIO1.SPCSO1.stVal")}
    command = {"ied_id": ied["id"], "ref": "TripCTRL/GGIO1.SPCSO1"}
    projects.diagram(diagram(node("TRIP_RELAY", state, command)))
    data = projects.project["diagram"]["nodes"][0]["data"]
    assert data["state_mapping"] == state
    assert data["control_binding"] == command
    assert data["trip_command_polarity"] == "operate_false"
    assert data["trip_state_polarity"] == "operated_true"


def test_push_button_modes_accept_only_their_relevant_mapping(tmp_path):
    projects = ProjectService(tmp_path)
    ied = projects.add_template("Button")[0]
    state = {"mode": "direct", "source": source(ied, "ButtonCTRL/XCBR1.Loc.stVal")}
    command = {"ied_id": ied["id"], "ref": "ButtonCTRL/GGIO1.SPCSO1"}
    projects.diagram(diagram(
        node("PUSH_BUTTON", control=command, key="momentary",
             push_mode="spc_momentary", push_value=False),
        node("PUSH_BUTTON", state=state, key="latching", push_mode="latching_status"),
        node("PUSH_BUTTON", state=state, key="resetting", push_mode="self_resetting",
             push_rest_value=True, push_reset_ms=500)))
    with pytest.raises(StudioError, match="Only a momentary command"):
        projects.diagram(diagram(node("PUSH_BUTTON", state=state, control=command,
                                      push_mode="latching_status")))
    with pytest.raises(StudioError, match="Only a momentary command"):
        projects.diagram(diagram(node("PUSH_BUTTON", state=state, control=command,
                                      push_mode="self_resetting")))
    with pytest.raises(StudioError, match="Only a latching or self-resetting"):
        projects.diagram(diagram(node("PUSH_BUTTON", state=state, control=command,
                                      push_mode="spc_momentary")))


def test_format_four_trip_relay_migrates_to_reset_high():
    legacy = diagram(node("TRIP_RELAY"))
    migrated, notices = migrate_diagram(legacy, [], 4)
    data = migrated["nodes"][0]["data"]
    assert data["trip_command_polarity"] == "operate_false"
    assert data["trip_state_polarity"] == "operated_true"
    assert notices == ["equipment: Trip command polarity changed to Operate=0 / Reset=1"]


def test_format_five_trip_polarity_migrates_with_exact_backup_and_notice(tmp_path):
    projects = ProjectService(tmp_path)
    projects.diagram(diagram(node("TRIP_RELAY", key="legacy-trip",
                                  trip_command_polarity="operate_true")))
    with zipfile.ZipFile(io.BytesIO(projects.export())) as archive:
        document = json.loads(archive.read("project.json"))
    document["format"] = 5
    legacy = document["diagram"]["nodes"][0]["data"]
    legacy["trip_polarity"] = legacy.pop("trip_command_polarity")
    legacy.pop("trip_state_polarity")
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("project.json", json.dumps(document))
    payload = stream.getvalue()
    projects.restore(payload)
    data = projects.project["diagram"]["nodes"][0]["data"]
    assert data["trip_command_polarity"] == "operate_false"
    assert data["trip_state_polarity"] == "operated_true"
    notice = projects.snapshot()["migration_notice"]
    assert notice["details"] == [
        "legacy-trip: Trip command polarity changed to Operate=0 / Reset=1"]
    assert Path(notice["backup"]).read_bytes() == payload


def test_format_six_preserves_command_and_adds_operated_high_state_with_backup(tmp_path):
    projects = ProjectService(tmp_path)
    projects.diagram(diagram(node("TRIP_RELAY", trip_command_polarity="operate_true"),
                             node("CB", key="breaker")))
    with zipfile.ZipFile(io.BytesIO(projects.export())) as archive:
        document = json.loads(archive.read("project.json"))
    document["format"] = 6
    for legacy in (item["data"] for item in document["diagram"]["nodes"]):
        legacy["trip_polarity"] = legacy.pop("trip_command_polarity")
        legacy.pop("trip_state_polarity")
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("project.json", json.dumps(document))
    payload = stream.getvalue()
    projects.restore(payload)
    data = projects.project["diagram"]["nodes"][0]["data"]
    assert data["trip_command_polarity"] == "operate_true"
    assert data["trip_state_polarity"] == "operated_true"
    assert "trip_polarity" not in projects.project["diagram"]["nodes"][1]["data"]
    notice = projects.snapshot()["migration_notice"]
    assert notice["details"] == [
        "equipment: Trip state polarity set to Operated=1 / Reset=0"]
    assert Path(notice["backup"]).read_bytes() == payload


def test_format_seven_preserves_trip_polarities_with_standard_commands(tmp_path):
    projects = ProjectService(tmp_path)
    model = projects.add_template("Seven")[0]
    command = {"ied_id": model["id"], "ref": "SevenCTRL/GGIO1.SPCSO1"}
    projects.diagram(diagram(node("TRIP_RELAY", control=command,
                                  trip_command_polarity="operate_true",
                                  trip_state_polarity="operated_false")))
    with zipfile.ZipFile(io.BytesIO(projects.export())) as archive:
        document = json.loads(archive.read("project.json"))
    document["format"] = 7
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("project.json", json.dumps(document))
    projects.restore(stream.getvalue())
    data = projects.project["diagram"]["nodes"][0]["data"]
    assert data["trip_command_polarity"] == "operate_true"
    assert data["trip_state_polarity"] == "operated_false"
    assert data["control_binding"] == command
    assert projects.snapshot()["migration_notice"]["details"] == [
        "Existing controls remain in standard IEC 61850 mode."]


def test_format_fourteen_concen_command_migrates_to_standard_with_notice(tmp_path):
    projects = ProjectService(tmp_path)
    model = projects.add_template("Compat")[0]
    command = {"ied_id": model["id"], "ref": "CompatCTRL/XCBR1.Pos"}
    projects.diagram(diagram(node("CB", control=command, key="cb1")))
    with zipfile.ZipFile(io.BytesIO(projects.export())) as archive:
        document = json.loads(archive.read("project.json"))
    document["format"] = 14
    document["diagram"]["nodes"][0]["data"]["control_binding"]["compatibility"] = "concen_inverted_sbow"
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("project.json", json.dumps(document))
    payload = stream.getvalue()
    projects.restore(payload)
    assert projects.project["diagram"]["nodes"][0]["data"]["control_binding"] == command
    notice = projects.snapshot()["migration_notice"]
    assert "format 15" in notice["message"] and "standard IEC 61850" in notice["message"]
    assert notice["details"] == ["cb1: the CONCEN command mode was removed; commands now follow "
                                 "standard IEC 61850 (Close = true, moves on Operate)"]
    assert Path(notice["backup"]).read_bytes() == payload


def test_legacy_boolean_mapping_migrates_and_loss_has_verified_backup(tmp_path):
    projects = ProjectService(tmp_path)
    ied = projects.add_template("Legacy")[0]
    with zipfile.ZipFile(io.BytesIO(projects.export())) as archive:
        document = json.loads(archive.read("project.json"))
    document["format"] = 3
    document["diagram"] = diagram(
        {"id": "kept", "position": {"x": 0, "y": 0}, "data": {"kind": "CB", "label": "Kept",
         "ied_id": ied["id"], "status_ref": "LegacyCTRL/XCBR1.Loc.stVal",
         "control_ref": "LegacyCTRL/XCBR1.Pos"}},
        {"id": "removed", "position": {"x": 1, "y": 1}, "data": {"kind": "DS", "label": "Removed",
         "ied_id": ied["id"], "status_ref": "LegacyCTRL/GGIO1.AnIn1.mag.f"}})
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("project.json", json.dumps(document))
    payload = stream.getvalue()
    projects.restore(payload)
    kept = projects.project["diagram"]["nodes"][0]["data"]
    assert kept["state_mapping"]["mode"] == "single_to_dbpos"
    assert kept["control_binding"]["ref"] == "LegacyCTRL/XCBR1.Pos"
    assert projects.project["diagram"]["nodes"][1]["data"]["state_mapping"] is None
    notice = projects.snapshot()["migration_notice"]
    assert "unsupported FLOAT32" in notice["details"][0]
    backup = Path(notice["backup"])
    assert backup.read_bytes() == payload
    with zipfile.ZipFile(backup) as archive:
        assert archive.testzip() is None


def test_format_nine_push_button_wires_are_removed_with_notice():
    buttons = diagram(node("PUSH_BUTTON", key="button"),
                      node("CB", key="breaker"))
    buttons["edges"] = [
        {"id": "w1", "source": "button", "target": "breaker"},
        {"id": "w2", "source": "breaker", "target": "button"},
        {"id": "w3", "source": "breaker", "target": "breaker"},
    ]
    migrated, notices = migrate_diagram(buttons, [], 9)
    assert [edge["id"] for edge in migrated["edges"]] == ["w3"]
    assert notices == ["2 wires to push buttons removed: buttons no longer accept connections"]
    assert migrate_diagram(migrated, [], 10) == (migrated, [])
