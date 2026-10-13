"""Saved scenario editing and explicit runtime sequence commands."""
from flask import Blueprint, jsonify, request
from .context import projects, runtime, require_stopped

blueprint = Blueprint("scenarios", __name__)


@blueprint.put("/scenarios")
def save_scenarios():
    require_stopped()
    projects().scenarios(request.get_json())
    return jsonify(ok=True)


@blueprint.get("/scenarios/runs")
def runs():
    return jsonify(runtime().scenario_runner.snapshot())


@blueprint.post("/scenarios/<key>/start")
def start(key):
    runtime().start_scenario(key)
    return jsonify(ok=True)
