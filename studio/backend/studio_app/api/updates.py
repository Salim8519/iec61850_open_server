"""Update check endpoints (ADR-0071): read the stored answer, check now, change settings."""
from flask import Blueprint, current_app, jsonify, request

blueprint = Blueprint("updates", __name__)


def _service():
    return current_app.extensions["updates"]


@blueprint.get("/app/update")
def status():
    return jsonify(_service().status())


@blueprint.post("/app/update/check")
def check_now():
    return jsonify(_service().check(manual=True))


@blueprint.put("/app/update")
def configure():
    try:
        return jsonify(_service().configure(request.get_json(silent=True)))
    except ValueError as error:
        return jsonify(error=str(error), code="invalid"), 400
