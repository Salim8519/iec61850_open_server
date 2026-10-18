"""Node-aware equipment operations backed by imported SCL control metadata."""
from flask import Blueprint, jsonify, request
from studio_app.simulation.equipment_test import apply_equipment_test
from .context import runtime

blueprint = Blueprint("equipment", __name__)


@blueprint.get("/equipment/states")
def states():
    return jsonify(states=runtime().equipment_states.snapshot(), energization=runtime().energization.snapshot())


@blueprint.post("/equipment/<node_id>/operate")
def operate(node_id):
    data = request.get_json(silent=True) or {}
    if not isinstance(data, dict):
        raise ValueError("Equipment operation must be a JSON object")
    return jsonify(runtime().equipment_control.operate(
        node_id, value=data.get("value"), action=data.get("action")))


@blueprint.post("/equipment/<node_id>/test")
def test(node_id):
    """Test button on a simulated IED: write the bound point, send no command (ADR-0067)."""
    data = request.get_json(silent=True) or {}
    if not isinstance(data, dict):
        raise ValueError("Equipment test must be a JSON object")
    return jsonify(apply_equipment_test(runtime(), node_id, data.get("action"), data.get("value"), data.get("row")))
