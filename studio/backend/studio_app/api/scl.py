"""SCL import, laboratory template creation, and original-file download."""
import io
import json
from flask import Blueprint, jsonify, request, send_file
from .context import projects, require_stopped
from studio_app.scl.document import preview_ieds
from studio_app.contracts.errors import StudioError
from studio_app.scl.schema_validation import validate_schema
from studio_app.scl.exporter import export_ied

blueprint = Blueprint("scl", __name__)


@blueprint.post("/scl/validate")
def validate_file():
    return jsonify(validate_schema(request.files["file"].read()))


@blueprint.post("/scl/preview")
def preview_file():
    require_stopped()
    upload = request.files["file"]
    models = preview_ieds(upload.read())
    existing = {ied["name"] for ied in projects().snapshot()["ieds"]}
    for model in models:
        if model["name"] in existing:
            model["unavailable_reason"] = "Already in this project"
    return jsonify(ieds=models, available_slots=max(0, 20 - len(existing)))


@blueprint.post("/scl/import")
def import_file():
    require_stopped()
    upload = request.files["file"]
    if "selected_names" not in request.form:
        raise StudioError("Preview the file and choose which IEDs to import first")
    selected = json.loads(request.form["selected_names"])
    if not isinstance(selected, list) or not 1 <= len(selected) <= 20:
        raise StudioError("Select between 1 and 20 IEDs")
    models = projects().add_scl(upload.read(), upload.filename or "import.scd", selected)
    return jsonify(ieds=[model["id"] for model in models])


@blueprint.post("/ieds/template")
def template():
    require_stopped()
    name = request.get_json().get("name", "BayController")
    models = projects().add_template(name)
    return jsonify(ieds=[model["id"] for model in models])


@blueprint.get("/ieds/<ied_id>/source")
def source(ied_id):
    ied = projects().find_ied(ied_id)
    return send_file(io.BytesIO(ied["source"].encode("utf-8")), mimetype="application/xml",
                     as_attachment=True, download_name=ied["filename"])


@blueprint.post("/ieds/<ied_id>/export-scl")
def edited_source(ied_id):
    payload = request.get_json()
    if not isinstance(payload, dict) or set(payload) != {"overrides"}:
        raise StudioError("Provide an overrides object; engineering addresses remain unchanged")
    ied = projects().find_ied(ied_id)
    document = export_ied(ied["source"], ied["name"], payload["overrides"])
    return send_file(io.BytesIO(document), mimetype="application/xml", as_attachment=True,
                     download_name=ied["name"] + ".edited.scd")
