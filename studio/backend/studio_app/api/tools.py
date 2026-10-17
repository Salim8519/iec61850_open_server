"""Tools place: IED Explorer and Find IEDs. Handlers validate and delegate only."""
import csv
import io
from flask import Blueprint, Response, current_app, jsonify, request

blueprint = Blueprint("tools", __name__)


def explorer():
    return current_app.extensions["explorer"]


def scanner():
    return current_app.extensions["scanner"]


def body():
    data = request.get_json(silent=True)
    if not isinstance(data, dict):
        raise ValueError("Send a JSON object")
    return data


def items(data):
    entries = data.get("items")
    if not isinstance(entries, list) or not all(isinstance(item, dict) and isinstance(item.get("ref"), str)
                                                and isinstance(item.get("fc"), str) for item in entries):
        raise ValueError("Send items as a list of {ref, fc}")
    return entries


@blueprint.get("/tools/explorer")
def explorer_status():
    return jsonify(explorer().warm())


@blueprint.post("/tools/explorer/connect")
def explorer_connect():
    data = body()
    return jsonify(explorer().connect(data.get("host", ""), data.get("port", 102)))


@blueprint.post("/tools/explorer/disconnect")
def explorer_disconnect():
    return jsonify(explorer().disconnect())


@blueprint.get("/tools/explorer/devices")
def explorer_devices():
    return jsonify(names=explorer().devices())


@blueprint.get("/tools/explorer/nodes")
def explorer_nodes():
    return jsonify(names=explorer().nodes(request.args.get("device", "")))


@blueprint.get("/tools/explorer/node")
def explorer_node():
    return jsonify(explorer().node(request.args.get("ref", "")))


@blueprint.post("/tools/explorer/read")
def explorer_read():
    return jsonify(values=explorer().read(items(body())))


@blueprint.put("/tools/explorer/watch")
def explorer_watch():
    data = body()
    return jsonify(explorer().set_watch(items(data), data.get("interval_ms")))


@blueprint.post("/tools/explorer/polling")
def explorer_polling():
    service = explorer()
    return jsonify(service.start_polling() if body().get("on") else service.stop_polling())


@blueprint.get("/tools/explorer/watch.csv")
def explorer_csv():
    status = explorer().status()
    stream = io.StringIO()
    writer = csv.writer(stream)
    writer.writerow(["reference", "fc", "value", "status", "read_at_unix"])
    for item in status["watch"]:
        writer.writerow([item["ref"], item["fc"], item.get("text", ""), item.get("status", ""), item.get("at") or ""])
    name = (status["host"] or "device").replace(".", "-")
    return Response(stream.getvalue(), mimetype="text/csv",
                    headers={"Content-Disposition": f"attachment; filename=watch-{name}.csv"})


@blueprint.post("/tools/scan/preview")
def scan_preview():
    data = body()
    return jsonify(scanner().preview(data.get("range", ""), data.get("port", 102)))


@blueprint.post("/tools/scan/start")
def scan_start():
    data = body()
    return jsonify(scanner().start(data.get("range", ""), data.get("port", 102)))


@blueprint.post("/tools/scan/stop")
def scan_stop():
    return jsonify(scanner().stop())


@blueprint.get("/tools/scan")
def scan_status():
    return jsonify(scanner().status())
