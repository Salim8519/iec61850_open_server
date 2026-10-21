"""Project and diagram HTTP endpoints."""
import io
from flask import Blueprint, current_app, jsonify, request, send_file
from .context import projects, runtime, require_stopped
from ..projects.project_files import DOWNLOAD_NAME, refuse_old_project

blueprint = Blueprint("projects", __name__)


@blueprint.get("/project")
def get_project():
    return jsonify({**projects().snapshot(), "runtime": runtime().state,
                    "running_ieds": sorted(runtime().active),
                    "startup_recovery": current_app.extensions["startup"].result})


@blueprint.put("/project/diagram")
def diagram():
    require_stopped()
    data = request.get_json(silent=True) or {}
    if set(data) != {"diagram", "revision", "tab_id"}:
        raise ValueError("Diagram update requires its project revision and tab identity")
    if (not isinstance(data["revision"], int) or not isinstance(data["tab_id"], str)
            or not 1 <= len(data["tab_id"]) <= 64):
        raise ValueError("Invalid project revision or tab identity")
    projects().diagram(data["diagram"], data["revision"], data["tab_id"])
    return jsonify(ok=True, revision=projects().revision)


@blueprint.post("/project/save")
def save():
    projects().save()
    return jsonify(ok=True)


@blueprint.get("/project/download")
def download():
    return send_file(io.BytesIO(projects().export()), mimetype="application/zip",
                     as_attachment=True, download_name=DOWNLOAD_NAME)


@blueprint.post("/project/open")
def open_project():
    require_stopped()
    upload = request.files["file"]
    refuse_old_project(upload.filename)  # a .subsim file is refused in plain words (ADR-0096)
    projects().restore(upload.read())
    return jsonify(projects().snapshot())
