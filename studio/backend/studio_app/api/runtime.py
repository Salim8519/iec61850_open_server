"""Runtime lifecycle commands."""
from flask import Blueprint, jsonify
from .context import runtime

blueprint = Blueprint("runtime", __name__)


@blueprint.post("/runtime/<action>")
def command(action):
    commands = {"start": runtime().start, "stop": runtime().stop, "pause": runtime().pause}
    if action not in commands:
        return jsonify(error="Unknown runtime action"), 404
    commands[action]()
    return jsonify(state=runtime().state)
