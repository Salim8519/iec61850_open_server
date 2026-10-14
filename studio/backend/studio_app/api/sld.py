"""Single-line diagram import endpoints (preview only; apply uses PUT /project/diagram)."""
from flask import Blueprint, jsonify, request
from studio_app.projects import sld_import
from .context import projects

blueprint = Blueprint("sld", __name__)


@blueprint.get("/project/substation-sources")
def sources():
    return jsonify(sources=sld_import.substation_sources(projects()))


@blueprint.post("/project/diagram/from-scl/preview")
def preview():
    data = request.get_json(silent=True) or {}
    if set(data) != {"source_id", "mode"} or not all(isinstance(data[key], str) for key in data):
        raise ValueError("Preview requires source_id and mode")
    return jsonify(sld_import.preview(projects(), data["source_id"], data["mode"]))
