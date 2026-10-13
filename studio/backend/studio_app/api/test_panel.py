"""Saved test-panel configuration and explicit runtime test actions."""
from flask import Blueprint, jsonify, request
from .context import projects, runtime, require_stopped

blueprint = Blueprint("test_panel", __name__)


@blueprint.put("/project/test-panel")
def save_panel():
    with runtime().cycle_lock:
        require_stopped()
        projects().test_panel(request.get_json())
        return jsonify(ok=True)


@blueprint.get("/project/test-panel")
def panel():
    c = runtime()
    with projects().lock:
        entries = list(projects().project["test_panel"])
    result = []
    for entry in entries:
        signal = c.signal(entry["ied_id"], entry["ref"])
        source = c.sources.get((entry["ied_id"], entry["ref"]), {})
        result.append({**entry, "signal": signal, "overridden": source.get("overridden", False),
                       "automatic": bool(source and source.get("mode") != "manual")})
    return jsonify(items=result)


@blueprint.post("/signals/pulse")
def pulse():
    data = request.get_json()
    return jsonify(runtime().pulses.start(data["ied_id"], data["ref"],
                                          data.get("active_value", True), data.get("duration_ms", 1000)))


@blueprint.post("/signals/release-override")
def release():
    data = request.get_json()
    runtime().release_override(data["ied_id"], data["ref"])
    return jsonify(ok=True)
