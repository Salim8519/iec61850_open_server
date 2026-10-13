"""Relay equipment is a named, expandable set of signal statuses."""
import pytest
from studio_app.contracts.errors import StudioError
from studio_app.projects.mapping_migration import migrate_diagram
from studio_app.projects.removal import impact, without_ieds
from studio_app.projects.validation import validate_diagram
from studio_app.simulation.equipment_states import EquipmentStates

IED = {"id": "ied-a", "signals": [
    {"ref": "GGIO1.Ind1.stVal", "kind": "BOOLEAN", "fc": "ST", "writable": True},
    {"ref": "GGIO1.Read.stVal", "kind": "BOOLEAN", "fc": "ST", "writable": False},
]}
IEDS = [IED]


def entry(entry_id="s1", label="Trip alarm", ied="ied-a", ref="GGIO1.Ind1.stVal"):
    return {"id": entry_id, "label": label,
            "source": {"ied_id": ied, "ref": ref, "report_ref": None}}


def test_relay_with_named_status_entries_is_accepted():
    result = validate_diagram(diagram_with([entry()]), [IED])
    saved = result["nodes"][0]["data"]["relay_statuses"]
    assert saved == [{"id": "s1", "label": "Trip alarm", "source": {"ied_id": "ied-a",
                     "ref": "GGIO1.Ind1.stVal", "report_ref": None}}]


def diagram_with(entries):
    return {"nodes": [{"id": "r1", "position": {"x": 0, "y": 0},
                       "data": {"kind": "RELAY", "label": "Relay 1", "rotation": 0,
                                "relay_statuses": entries}}], "edges": []}


def test_non_writable_signal_is_accepted_as_display_only_status():
    validate_diagram(diagram_with([entry(ref="GGIO1.Read.stVal")]), [IED])


def test_status_entry_must_reference_a_known_signal():
    with pytest.raises(StudioError, match="unknown signal"):
        validate_diagram(diagram_with([entry(ref="Missing.stVal")]), [IED])
    with pytest.raises(StudioError, match="unknown IED"):
        validate_diagram(diagram_with([entry(ied="missing")]), [IED])


def test_relay_rejects_the_legacy_single_state_mapping():
    legacy = {"nodes": [{"id": "r1", "position": {"x": 0, "y": 0},
                         "data": {"kind": "RELAY", "label": "Relay 1", "rotation": 0,
                                  "state_mapping": {"mode": "direct",
                                                    "source": {"ied_id": "ied-a", "ref": "GGIO1.Ind1.stVal"}}}}],
              "edges": []}
    with pytest.raises(StudioError, match="named status entries"):
        validate_diagram(legacy, [IED])


def test_relay_statuses_are_rejected_on_other_equipment():
    breaker = {"nodes": [{"id": "cb", "position": {"x": 0, "y": 0},
                          "data": {"kind": "CB", "label": "CB 1", "rotation": 0,
                                   "relay_statuses": [entry()]}}], "edges": []}
    with pytest.raises(StudioError, match="Only a relay"):
        validate_diagram(breaker, [IED])


def test_duplicate_status_identifiers_are_rejected():
    with pytest.raises(StudioError, match="unique"):
        validate_diagram(diagram_with([entry(), entry()]), [IED])


def test_format_8_relay_state_mapping_becomes_one_named_status():
    legacy = {"nodes": [{"id": "r1", "position": {"x": 0, "y": 0},
                         "data": {"kind": "RELAY", "label": "Relay 1", "rotation": 0,
                                  "state_mapping": {"mode": "direct",
                                                    "source": {"ied_id": "ied-a", "ref": "GGIO1.Ind1.stVal"}}}}],
              "edges": []}
    migrated, notices = migrate_diagram(legacy, [IED], 8)
    data = migrated["nodes"][0]["data"]
    assert data["relay_statuses"] == [{"id": "s1", "label": "Relay 1",
                                       "source": {"ied_id": "ied-a", "ref": "GGIO1.Ind1.stVal"}}]
    assert "state_mapping" not in data
    assert any("Relay 1" in notice for notice in notices)
    assert migrate_diagram(legacy, [IED], 9)[0] == legacy


def test_removal_counts_and_strips_relay_status_bindings():
    project = {"ieds": [{"id": "ied-a"}, {"id": "ied-b"}],
               "diagram": {"nodes": [{"id": "r1", "data": {"kind": "RELAY",
               "relay_statuses": [entry(), entry(ied="ied-b")]}}, {"id": "cb", "data": {"kind": "CB"}}],
               "edges": []}, "scenarios": [], "sources": [], "test_panel": []}
    assert impact(project, "ied-a")["bindings"] == 1
    cleaned = without_ieds(project, {"ied-a"})
    kept = cleaned["diagram"]["nodes"][0]["data"]["relay_statuses"]
    assert kept == [entry(ied="ied-b")]


class FakeCoordinator:
    def __init__(self, signals, state="running"):
        self.index = signals
        self.state = state
        self.events = []

    def emit(self, kind, **payload):
        self.events.append((kind, payload))


def test_relay_statuses_project_as_composite_node_keys():
    coordinator = FakeCoordinator({("ied-a", "GGIO1.Ind1.stVal"): {"value": True}})
    states = EquipmentStates(coordinator)
    diagram = {"nodes": [{"id": "r1", "data": {"kind": "RELAY", "relay_statuses": [entry()]}}], "edges": []}
    states.reset(diagram)
    states.publish_all()
    assert ("equipment_state", {"node_id": "r1:s1", "value": True}) in coordinator.events
    snapshot = states.snapshot()
    assert snapshot == [{"node_id": "r1:s1", "value": True}]
