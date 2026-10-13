import pytest
from pydantic import ValidationError
from studio_app.contracts.errors import StudioError
from studio_app.contracts.project import Diagram
from studio_app.projects.mapping_migration import migrate_diagram
from studio_app.projects.validation import validate_diagram


def junction(node_id="j1", **extra):
    data = {"label": "J1", "kind": "JUNCTION", "rotation": 0}
    data.update(extra)
    return {"id": node_id, "position": {"x": 0, "y": 0}, "data": data}


def test_junction_wires_keep_their_terminal_sides():
    diagram = {"nodes": [junction(), junction("j2")],
               "edges": [{"id": "w1", "source": "j1", "target": "j2",
                          "sourceHandle": "east", "targetHandle": "north"}]}
    result = validate_diagram(diagram, [])
    assert result["nodes"][0]["data"]["kind"] == "JUNCTION"
    assert result["edges"][0]["sourceHandle"] == "east"


@pytest.mark.parametrize("kind", ["BUSBAR", "LOAD", "TRANSFORMER"])
def test_retired_kinds_are_rejected(kind):
    diagram = {"nodes": [{"id": "x", "position": {"x": 0, "y": 0},
                          "data": {"label": "X", "kind": kind, "rotation": 0}}], "edges": []}
    with pytest.raises(ValidationError):
        Diagram.model_validate(diagram)


def test_drawn_width_is_no_longer_a_field():
    diagram = {"nodes": [{"id": "cb", "position": {"x": 0, "y": 0},
                          "data": {"label": "CB1", "kind": "CB", "rotation": 0, "width": 300}}],
               "edges": []}
    with pytest.raises(ValidationError):
        Diagram.model_validate(diagram)


def test_junction_does_not_accept_a_state_mapping():
    binding = {"ied_id": "ied-a", "ref": "GGIO1.Ind1.stVal"}
    diagram = {"nodes": [junction(state_mapping={"mode": "direct", "source": binding})], "edges": []}
    with pytest.raises(StudioError, match="junction is a drawing point"):
        validate_diagram(diagram, [])


def test_junction_does_not_accept_a_control_binding():
    diagram = {"nodes": [junction(control_binding={"ied_id": "ied-a", "ref": "GGIO1.SPCSO1"})], "edges": []}
    with pytest.raises(StudioError, match="junction is a drawing point"):
        validate_diagram(diagram, [])


def test_format_12_migration_retires_legacy_kinds_and_their_wires():
    legacy = {"nodes": [junction(), {"id": "bus", "position": {"x": 0, "y": 0},
                                     "data": {"label": "Main bus", "kind": "BUSBAR", "rotation": 0, "width": 400}}],
              "edges": [{"id": "w1", "source": "j1", "target": "bus"}]}
    migrated, notices = migrate_diagram(legacy, [], 11)
    assert [node["id"] for node in migrated["nodes"]] == ["j1"] and migrated["edges"] == []
    assert notices == ["Main bus: old busbar symbol removed (no longer supported); its wires were removed too"]
    assert validate_diagram(migrated, [])["nodes"][0]["data"]["kind"] == "JUNCTION"
    assert migrate_diagram(migrated, [], 12) == (migrated, [])
