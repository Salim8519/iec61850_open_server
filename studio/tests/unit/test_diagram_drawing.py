import pytest
from pydantic import ValidationError
from studio_app.contracts.errors import StudioError
from studio_app.contracts.project import Diagram
from studio_app.projects.validation import validate_diagram


def junction(node_id="j1", **extra):
    data = {"label": "J1", "kind": "JUNCTION", "rotation": 0}
    data.update(extra)
    return {"id": node_id, "position": {"x": 0, "y": 0}, "data": data}


def test_junction_and_drawn_busbar_are_accepted():
    diagram = {"nodes": [junction(),
                         {"id": "bus", "position": {"x": 0, "y": 200},
                          "data": {"label": "B1", "kind": "BUSBAR", "rotation": 0, "width": 400}}],
               "edges": [{"id": "w1", "source": "j1", "target": "bus",
                          "sourceHandle": "east", "targetHandle": "north"}]}
    result = validate_diagram(diagram, [])
    assert result["nodes"][0]["data"]["kind"] == "JUNCTION"
    assert result["nodes"][1]["data"]["width"] == 400
    assert result["edges"][0]["sourceHandle"] == "east"


@pytest.mark.parametrize("value", [59, 2401])
def test_busbar_width_outside_draw_bounds_is_rejected(value):
    diagram = {"nodes": [{"id": "bus", "position": {"x": 0, "y": 0},
                          "data": {"label": "B1", "kind": "BUSBAR", "rotation": 0, "width": value}}],
               "edges": []}
    with pytest.raises(ValidationError):
        Diagram.model_validate(diagram)


def test_drawn_width_is_rejected_on_non_busbar_equipment():
    diagram = {"nodes": [{"id": "cb", "position": {"x": 0, "y": 0},
                          "data": {"label": "CB1", "kind": "CB", "rotation": 0, "width": 300}}],
               "edges": []}
    with pytest.raises(StudioError, match="Only a busbar accepts a drawn width"):
        validate_diagram(diagram, [])


def test_junction_does_not_accept_a_state_mapping():
    binding = {"ied_id": "ied-a", "ref": "GGIO1.Ind1.stVal"}
    diagram = {"nodes": [junction(state_mapping={"mode": "direct", "source": binding})], "edges": []}
    with pytest.raises(StudioError, match="junction is a drawing point"):
        validate_diagram(diagram, [])


def test_junction_does_not_accept_a_control_binding():
    diagram = {"nodes": [junction(control_binding={"ied_id": "ied-a", "ref": "GGIO1.SPCSO1"})], "edges": []}
    with pytest.raises(StudioError, match="junction is a drawing point"):
        validate_diagram(diagram, [])


def test_legacy_diagram_without_width_stays_valid():
    diagram = {"nodes": [{"id": "bus", "position": {"x": 0, "y": 0},
                          "data": {"label": "B1", "kind": "BUSBAR", "rotation": 0}}], "edges": []}
    assert validate_diagram(diagram, [])["nodes"][0]["data"]["width"] is None
