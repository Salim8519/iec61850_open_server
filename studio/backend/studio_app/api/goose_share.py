"""Where GOOSE goes: preview, switch sharing on or off, and answer conflicts (ADR-0073)."""
from flask import Blueprint, jsonify, request
from .context import projects, runtime

blueprint = Blueprint("goose_share", __name__)


def body():
    return request.get_json(silent=True) or {}


@blueprint.get("/goose/share")
def preview():
    return jsonify(runtime().live.share.preview(projects().project))


@blueprint.put("/goose/share")
def apply():
    return jsonify(runtime().live.share.apply(projects(), body()))


@blueprint.post("/goose/conflicts")
def answer():
    return jsonify(runtime().live.share.choose(body()))
