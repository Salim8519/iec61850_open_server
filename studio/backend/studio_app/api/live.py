"""Real-IED status, SCD comparison, on-demand reads and GOOSE traffic."""
import csv
import io
from flask import Blueprint, Response, jsonify, request
from .context import projects, runtime
from studio_app.domain.goose_supervision import watch_list
from studio_app.domain.sv_supervision import watch_list as sv_watch_list
from studio_app.networking.npcap import npcap_status
from studio_app.projects.goose_links import link_map

blueprint = Blueprint("live", __name__)


@blueprint.get("/live")
def status():
    return jsonify(runtime().live.status())


def _comparison(key):
    comparison = runtime().live.session(key).status.get("comparison")
    if comparison is None:
        raise ValueError("The comparison is ready after ScadaBench has connected to the device and read its model")
    return comparison


@blueprint.get("/live/<key>/comparison")
def comparison(key):
    return jsonify(_comparison(key))


@blueprint.get("/live/<key>/comparison.csv")
def comparison_csv(key):
    stream = io.StringIO()
    writer = csv.writer(stream)
    writer.writerow(["reference", "fc", "result", "scd_type", "device_type"])
    for row in _comparison(key)["signals"]:
        writer.writerow([row["ref"], row["fc"], row["category"], row["scd_type"] or "", row["device_type"] or ""])
    name = projects().find_ied(key)["name"]
    return Response(stream.getvalue(), mimetype="text/csv",
                    headers={"Content-Disposition": f"attachment; filename={name}-scd-vs-device.csv"})


@blueprint.post("/live/<key>/read")
def read(key):
    refs = (request.get_json(silent=True) or {}).get("refs")
    if not isinstance(refs, list) or not refs or len(refs) > 500:
        raise ValueError("Send between 1 and 500 references to read")
    ied = projects().find_ied(key)
    by_ref = {signal["ref"]: signal for signal in ied["signals"]}
    items = [{"ref": ref, "fc": by_ref[ref]["fc"]} for ref in refs if ref in by_ref]
    result = runtime().live.session(key).read(items)
    index = runtime().ied_indexes.get(key, {})
    values = {ref: {"value": index[ref]["value"], "live_state": index[ref].get("live_state")}
              for ref in refs if ref in index}
    return jsonify(values=values, errors=result["errors"])


@blueprint.get("/goose")
def goose():
    if runtime().state != "stopped":
        return jsonify(running=True, npcap=npcap_status(), **runtime().live.goose.snapshot())
    watches = list(watch_list(projects().project["ieds"]).values())
    for watch in watches:
        watch.update(state="stopped", problem="", frames=0)
    return jsonify(running=False, npcap=npcap_status(), state="stopped", unknown=[], watches=watches,
                   message="Press Run to start listening for GOOSE")


@blueprint.get("/goose/links")
def goose_links():
    running = runtime().state != "stopped"
    watches = runtime().live.goose.snapshot()["watches"] if running else []
    return jsonify(link_map(projects(), watches, running))


@blueprint.get("/mms/access")
def mms_access():
    return jsonify(running=runtime().state != "stopped", ieds=runtime().mms_access.snapshot())


@blueprint.get("/sv")
def sampled_values():
    if runtime().state != "stopped":
        return jsonify(running=True, npcap=npcap_status(), **runtime().live.sv.snapshot())
    network = projects().project["network"]
    supervised, extra = sv_watch_list(projects().project["ieds"], network.get("nominal_hz", 50))
    return jsonify(running=False, npcap=npcap_status(), state="stopped", unknown=[], watches=supervised,
                   not_supervised=[{"key": item["key"], "publisher": item["publisher"]} for item in extra],
                   nominal_hz=network.get("nominal_hz", 50), message="Press Run to start listening for Sampled Values")
