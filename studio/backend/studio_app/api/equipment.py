"""Node-aware equipment operations backed by imported SCL control metadata."""
from flask import Blueprint, jsonify, request
from .context import runtime

blueprint = Blueprint("equipment", __name__)


@blueprint.get("/equipment/states")
def states():
    return jsonify(states=runtime().equipment_states.snapshot())


@blueprint.post("/equipment/<node_id>/operate")
def operate(node_id):
    data = request.get_json(silent=True) or {}
    if not isinstance(data, dict):
        raise ValueError("Equipment operation must be a JSON object")
    return jsonify(runtime().equipment_control.operate(
        node_id, value=data.get("value"), action=data.get("action")))
