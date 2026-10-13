"""Lab network mode, adapter preview/apply/cleanup and per-IED network settings."""
from flask import Blueprint, current_app, jsonify, request
from .context import projects, runtime, require_stopped
from studio_app.networking.npcap import npcap_status
from studio_app.projects.ied_settings import update_ied, update_network

blueprint = Blueprint("network", __name__)


def network():
    return current_app.extensions["network"]


def body():
    return request.get_json(silent=True) or {}


@blueprint.get("/network")
def status():
    return jsonify(network=projects().project["network"], npcap=npcap_status())


@blueprint.get("/network/adapters")
def adapters():
    return jsonify(adapters=network().adapters(refresh=request.args.get("refresh") == "1"))


@blueprint.get("/network/leftovers")
def leftovers():
    return jsonify(items=network().leftovers())


@blueprint.put("/network")
def mode():
    with runtime().cycle_lock:
        require_stopped()
        return jsonify(network=update_network(projects(), body()))


@blueprint.post("/network/plan/preview")
def preview():
    data = body()
    require_stopped()
    return jsonify(network().preview(data.get("adapter_guid"), data.get("strategy", "scd"), data.get("manual")))


@blueprint.post("/network/plan/apply")
def apply():
    data = body()
    with runtime().cycle_lock:
        require_stopped()
        return jsonify(network().apply(data.get("fingerprint"), data.get("confirm")))


@blueprint.get("/network/cleanup/preview")
def cleanup_preview():
    return jsonify(network().cleanup_preview())


@blueprint.post("/network/cleanup/apply")
def cleanup():
    data = body()
    with runtime().cycle_lock:
        require_stopped()
        return jsonify(network().cleanup(data.get("fingerprint"), data.get("confirm")))


@blueprint.put("/ieds/<key>/settings")
def ied_settings(key):
    with runtime().cycle_lock:
        require_stopped()
        return jsonify(update_ied(projects(), key, body()))
