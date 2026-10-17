"""Gateway simulator API (Tools place). Handlers validate input and delegate."""
from flask import Blueprint, Response, current_app, jsonify, request
from studio_app.domain.gateway_csv import export_csv, import_csv
from studio_app.domain.gateway_points import PROTOCOLS, propose
from studio_app.domain.gateway_rules import listen_address, validate_points, validate_settings
from .context import projects

blueprint = Blueprint("gateway", __name__)


def gateway():
    return current_app.extensions["gateway"]


def protocol_arg(value):
    if value not in PROTOCOLS:
        raise ValueError("Choose IEC 104 (iec104), Modbus TCP (modbus) or DNP3 (dnp3)")
    return value


def body():
    data = request.get_json(silent=True)
    if not isinstance(data, dict):
        raise ValueError("Send a JSON object")
    return data


@blueprint.get("/tools/gateway")
def status():
    return jsonify(gateway().status())


@blueprint.get("/tools/gateway/config")
def config():
    current = gateway().config()
    return jsonify(protocol=current["protocol"], listen=current["listen"], settings=current["settings"],
                   real_commands=current.get("real_commands", True), auto_start=current.get("auto_start", False))


@blueprint.put("/tools/gateway/config")
def save_config():
    """Protocol, listen choice, real-IED commands and settings; the point list is saved separately."""
    data = body()
    service = gateway()
    if service.state == "running":
        raise ValueError("Stop the gateway before changing its settings")
    protocol = protocol_arg(data.get("protocol"))
    listen_address(data.get("listen", "this_pc"))
    current = service.config()
    settings = dict(current["settings"], **{protocol: validate_settings(protocol, data.get("settings"))})
    real_commands = data.get("real_commands", current.get("real_commands", True))
    if not isinstance(real_commands, bool):
        raise ValueError("Commands to real IEDs are on or off")
    projects().gateway({**current, "protocol": protocol, "listen": data.get("listen", "this_pc"), "settings": settings,
                        "real_commands": real_commands})
    return config()


@blueprint.get("/tools/gateway/points")
def points():
    return jsonify(points=gateway().table(protocol_arg(request.args.get("protocol"))))


@blueprint.put("/tools/gateway/points")
def save_points():
    data = body()
    service = gateway()
    if service.state == "running":
        raise ValueError("Stop the gateway before changing its point list")
    protocol = protocol_arg(data.get("protocol"))
    clean, problems = validate_points(protocol, data.get("points"), projects().project["ieds"])
    if problems:
        return jsonify(error="Some points need attention before they can be saved", problems=problems), 400
    current = service.config()
    projects().gateway({**current, "points": dict(current["points"], **{protocol: clean})})
    return jsonify(points=service.table(protocol))


def named(points):
    names = {ied["id"]: ied["name"] for ied in projects().project["ieds"]}
    roles = {ied["id"]: ied.get("role") for ied in projects().project["ieds"]}
    return [{**point, "ied_name": names.get(point["ied"], "?"), "monitor_only": roles.get(point["ied"]) != "simulated"}
            for point in points]


@blueprint.post("/tools/gateway/propose")
def propose_points():
    proposal, notes = propose(projects().project["ieds"], protocol_arg(body().get("protocol")))
    return jsonify(points=named(proposal), notes=notes)


@blueprint.get("/tools/gateway/points.csv")
def export_points():
    protocol = protocol_arg(request.args.get("protocol"))
    text = export_csv(gateway().config()["points"].get(protocol, []), projects().project["ieds"])
    return Response(text, mimetype="text/csv",
                    headers={"Content-Disposition": f"attachment; filename=gateway-{protocol}-points.csv"})


@blueprint.post("/tools/gateway/points.csv")
def import_points():
    """Parse a CSV point list for review; nothing is saved until the user saves the table."""
    data = body()
    protocol = protocol_arg(data.get("protocol"))
    parsed, problems = import_csv(str(data.get("text", "")), projects().project["ieds"])
    clean, invalid = validate_points(protocol, parsed, projects().project["ieds"])
    return jsonify(points=named(clean), problems=problems + invalid)


@blueprint.post("/tools/gateway/start")
def start():
    return jsonify(gateway().start())


@blueprint.post("/tools/gateway/stop")
def stop():
    return jsonify(gateway().stop())


@blueprint.get("/tools/gateway/traffic")
def traffic():
    return jsonify(gateway().traffic.after(int(request.args.get("after", 0))))


@blueprint.get("/tools/gateway/commands")
def commands():
    return jsonify(gateway().commands.after(int(request.args.get("after", 0)), limit=200))
