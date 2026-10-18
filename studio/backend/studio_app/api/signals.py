"""Paged signal inspection and typed runtime updates."""
from flask import Blueprint, current_app, jsonify, request
from studio_app.runtime.mms_client import operate
from studio_app.scl.report_catalogue import report_members
from .context import projects, runtime

blueprint = Blueprint("signals", __name__)


@blueprint.get("/ieds/<ied_id>/signals")
def list_signals(ied_id):
    ied = projects().find_ied(ied_id)
    query = request.args.get("q", "").casefold()
    fc = request.args.get("fc")
    kinds = {item for item in request.args.get("kind", "").split(",") if item}
    writable = request.args.get("writable") == "true"
    report = request.args.get("report")
    members = report_members(ied, report) if report else None
    exact = request.args.get("ref")
    # Read-only display filters of the Data points tab (change 0093); they never change a value.
    device = request.args.get("ld")
    bad_only = request.args.get("bad_quality") == "true"
    if request.args.get("in_dataset") == "true":
        in_sets = {ref for dataset in ied.get("datasets", []) for ref in dataset["signal_refs"]}
        members = in_sets if members is None else members & in_sets
    index = {signal["ref"]: signal for signal in ied["signals"]}
    bad = lambda signal: index.get(signal.get("quality_ref") or "", {}).get("value", 0) not in (0, None)
    filtered = [signal for signal in ied["signals"]
                if query in (signal["ref"] + " " + signal["description"]).casefold()
                and (not fc or signal["fc"] == fc) and (not exact or signal["ref"] == exact)
                and (not kinds or signal["kind"] in kinds)
                and (not writable or signal["writable"] and signal["fc"] != "CF")
                and (members is None or signal["ref"] in members)
                and (not device or signal["ref"].split("/", 1)[0] == device)
                and (not bad_only or bad(signal))]
    offset = max(0, int(request.args.get("offset", 0)))
    limit = min(250, max(1, int(request.args.get("limit", 100))))
    devices = sorted({signal["ref"].split("/", 1)[0] for signal in ied["signals"] if "/" in signal["ref"]})
    items = [{**signal, "quality": index.get(signal.get("quality_ref"), {}).get("value", 0),
              "timestamp": index.get(signal.get("timestamp_ref"), {}).get("value", 0)}
             for signal in filtered[offset:offset + limit]]
    return jsonify(total=len(filtered), items=items, devices=devices)


@blueprint.post("/signals/update")
def update():
    data = request.get_json()
    return jsonify(value=runtime().update(data["ied_id"], data["ref"], data["value"], quality=data.get("quality")))


@blueprint.post("/signals/source")
def configure_source():
    data = request.get_json()
    runtime().configure_source(data["ied_id"], data["ref"], data["spec"])
    return jsonify(ok=True)


@blueprint.post("/controls/operate")
def operate_control():
    data = request.get_json()
    ied = projects().find_ied(data["ied_id"])
    result = operate(current_app.extensions["studio_settings"].native_executable,
                     ied, data["ref"], data["value"])
    return jsonify(result)
