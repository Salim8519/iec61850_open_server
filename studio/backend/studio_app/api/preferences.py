"""UI preferences endpoints (guided tour state)."""
from flask import Blueprint, current_app, jsonify, request

blueprint = Blueprint("preferences", __name__)


@blueprint.get("/preferences")
def get_preferences():
    return jsonify(current_app.extensions["preferences"].get())


@blueprint.put("/preferences")
def put_preferences():
    return jsonify(current_app.extensions["preferences"].update(request.get_json(silent=True)))
