"""State colours are optional, validated, display-only equipment data."""
import pytest
from pydantic import ValidationError
from studio_app.contracts.project import EquipmentData
from studio_app.projects.service import ProjectService


def node(key, **extra):
    return {"id": key, "position": {"x": 0, "y": 0},
            "data": {"kind": "CB", "label": key, **extra}}


def test_colours_default_to_none_for_existing_projects():
    data = EquipmentData(kind="CB", label="CB1")
    assert data.active_color is None and data.inactive_color is None


def test_valid_hex_colours_are_kept_exactly():
    data = EquipmentData(kind="TRIP_RELAY", label="T1", active_color="#DC2626", inactive_color="#ffffff")
    assert (data.active_color, data.inactive_color) == ("#DC2626", "#ffffff")


@pytest.mark.parametrize("value", ["green", "#fff", "#12345G", "#1234567", "rgb(0,0,0)", ""])
def test_invalid_colours_are_rejected(value):
    with pytest.raises(ValidationError):
        EquipmentData(kind="CB", label="CB1", active_color=value)


def test_colours_are_stored_with_the_diagram(tmp_path):
    projects = ProjectService(tmp_path)
    projects.diagram({"nodes": [node("cb1", active_color="#16a34a", inactive_color="#f8fafc")],
                      "edges": []})
    saved = projects.snapshot()["diagram"]["nodes"][0]["data"]
    assert (saved["active_color"], saved["inactive_color"]) == ("#16a34a", "#f8fafc")
