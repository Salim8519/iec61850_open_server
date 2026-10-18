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
    return jsonify(state=runtime().state, running=sorted(runtime().active))


@blueprint.post("/runtime/ieds/<ied_id>/<action>")
def ied_command(ied_id, action):
    """Run or stop one IED; the other IEDs keep their state (ADR-0067)."""
    commands = {"start": runtime().start, "stop": runtime().stop}
    if action not in commands:
        return jsonify(error="Unknown runtime action"), 404
    commands[action]([ied_id])
    return jsonify(state=runtime().state, running=sorted(runtime().active))
