"""Mode, behaviour and simulation settings of ScadaBench's simulated IEDs."""
from flask import Blueprint, jsonify, request
from studio_app.simulation.test_mode_apply import set_test_mode
from .context import runtime

blueprint = Blueprint("test_mode", __name__)


@blueprint.put("/ieds/<key>/test-mode")
def update(key):
    return jsonify(set_test_mode(runtime(), key, request.get_json(silent=True) or {}))
