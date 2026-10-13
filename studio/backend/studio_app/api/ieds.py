"""IED deletion with impact preview and preserved recovery archive."""
from flask import Blueprint, jsonify, request
from .context import projects, runtime, require_stopped
from studio_app.projects import bulk_removal

blueprint = Blueprint("ieds", __name__)


@blueprint.get("/ieds/<key>/removal-preview")
def preview(key):
    require_stopped()
    return jsonify(projects().removal_preview(key))


@blueprint.delete("/ieds/<key>")
def remove(key):
    with runtime().cycle_lock:
        require_stopped()
        if (request.get_json(silent=True) or {}).get("confirm") is not True:
            return jsonify(error="Confirm deletion after reviewing affected bindings and automation"), 400
        return jsonify(projects().remove_ied(key))


@blueprint.post("/ieds/removal-preview")
def bulk_preview():
    with runtime().cycle_lock:
        require_stopped()
        return jsonify(bulk_removal.preview(projects(), request.get_json().get("ids")))


@blueprint.post("/ieds/remove")
def bulk_remove():
    with runtime().cycle_lock:
        require_stopped()
        data = request.get_json()
        if data.get("confirm") is not True:
            return jsonify(error="Confirm the deletion preview first"), 400
        result = bulk_removal.remove(projects(), data.get("ids"), data.get("fingerprint"))
        runtime().emit("ieds_removed", ied_ids=data["ids"], count=len(data["ids"]))
        return jsonify(result)
