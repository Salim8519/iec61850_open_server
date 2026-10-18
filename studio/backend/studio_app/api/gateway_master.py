"""SCADA master API (Tools place, "SCADA gateway & master"). Handlers validate input and delegate."""
from flask import Blueprint, Response, current_app, jsonify, request
from studio_app.domain.master_points import export_csv, import_labels
from studio_app.domain.master_rules import (protocol_arg, ranges_from_probe, validate_host, validate_probe, validate_ranges,
                                              validate_role, validate_routes, validate_settings)
from .context import projects

blueprint = Blueprint("gateway_master", __name__)
KINDS = {"iec104": {"single", "double", "setpoint_float", "setpoint_scaled", "setpoint_normalized", "clock"},
         "dnp3": {"binary_output", "analog_output_int", "analog_output_float"},
         "modbus": {"coil", "register", "registers_int32", "registers_float"}}
CROB_CODES = {"latch", "pulse", "trip_close"}


def master():
    return current_app.extensions["master"]


def body():
    data = request.get_json(silent=True)
    if not isinstance(data, dict):
        raise ValueError("Send a JSON object")
    return data


def idle():
    if master().process is not None or current_app.extensions["gateway"].state == "running":
        raise ValueError("Stop the gateway and disconnect the master before changing this")


@blueprint.post("/tools/gateway/role")
def choose_role():
    """Station gateway, SCADA master or protocol converter; one role runs at a time."""
    role = validate_role(body().get("role"))
    idle()
    projects().gateway({**projects().project["gateway"], "role": role})
    return jsonify(role=role)


@blueprint.get("/tools/master")
def status():
    return jsonify(master().status())


@blueprint.get("/tools/master/config")
def config():
    current = master().config()
    return jsonify(role=projects().project["gateway"].get("role", "gateway"),
                   **{key: current[key] for key in ("protocol", "host", "settings", "modbus_ranges", "routes", "auto_connect")})


@blueprint.put("/tools/master/config")
def save_config():
    data = body()
    if master().process is not None:
        raise ValueError("Disconnect before changing the connection settings")
    current = master().config()
    protocol = protocol_arg(data.get("protocol", current["protocol"]))
    host = validate_host(data["host"]) if data.get("host") else ""
    settings = dict(current["settings"], **{protocol: validate_settings(protocol, data.get("settings"))})
    ranges = validate_ranges(data.get("modbus_ranges", current["modbus_ranges"]))
    master().save({**current, "protocol": protocol, "host": host, "settings": settings, "modbus_ranges": ranges})
    return config()


@blueprint.post("/tools/master/connect")
def connect():
    return jsonify(master().connect())


@blueprint.post("/tools/master/disconnect")
def disconnect():
    return jsonify(master().disconnect())


@blueprint.post("/tools/master/interrogate")
def interrogate():
    return jsonify(master().interrogate())


@blueprint.get("/tools/master/points")
def points():
    return jsonify(points=master().points())


@blueprint.put("/tools/master/selection")
def selection():
    """Untick points the user does not want to watch; they stay discovered."""
    data = body()
    unticked = data.get("unticked")
    if not isinstance(unticked, list) or not all(isinstance(key, str) and len(key) <= 40 for key in unticked):
        raise ValueError("Send the unticked point keys as a list")
    current = master().config()
    protocol = current["protocol"]
    master().save({**current, "unticked": dict(current["unticked"], **{protocol: unticked[:20000]})})
    return jsonify(unticked=unticked[:20000])


@blueprint.put("/tools/master/label")
def label():
    data = body()
    key, text = data.get("key"), data.get("label", "")
    if not isinstance(key, str) or len(key) > 40 or not isinstance(text, str):
        raise ValueError("Send the point key and its name")
    current = master().config()
    labels = dict(current["labels"][current["protocol"]])
    if text.strip():
        labels[key] = text.strip()[:80]
    else:
        labels.pop(key, None)
    master().save({**current, "labels": dict(current["labels"], **{current["protocol"]: labels})})
    return jsonify(key=key, label=labels.get(key, ""))


@blueprint.get("/tools/master/points.csv")
def export_points():
    protocol = master().config()["protocol"]
    return Response(export_csv(master().points()), mimetype="text/csv",
                    headers={"Content-Disposition": f"attachment; filename=master-{protocol}-points.csv"})


@blueprint.post("/tools/master/labels.csv")
def import_point_labels():
    labels, problems = import_labels(str(body().get("text", "")))
    current = master().config()
    merged = dict(current["labels"][current["protocol"]], **labels)
    master().save({**current, "labels": dict(current["labels"], **{current["protocol"]: merged})})
    return jsonify(imported=len(labels), problems=problems)


@blueprint.post("/tools/master/command")
def command():
    """Sent only after the page's warning was confirmed (owner policy: real commands allowed)."""
    data = body()
    if data.get("confirm") is not True:
        raise ValueError("Confirm the command first")
    kind = data.get("kind")
    protocol = master().config()["protocol"]
    if kind not in KINDS[protocol]:
        raise ValueError(f"This command type is not available for {protocol}: choose one of {', '.join(sorted(KINDS[protocol]))}")
    if kind == "clock":
        return jsonify(master().command({"kind": kind}))
    address, value = data.get("address"), data.get("value")
    if isinstance(address, bool) or not isinstance(address, int) or not 0 <= address <= (16777215 if protocol == "iec104" else 65535):
        raise ValueError("The address must be a whole number")
    if isinstance(value, bool):
        value = int(value) if kind == "single" else value
    if not isinstance(value, (int, float)):
        raise ValueError("The value must be a number")
    command = {"kind": kind, "address": address, "value": value, "select": data.get("select") is True}
    if kind.startswith("registers_"):
        command["word_order"] = "little" if data.get("word_order") == "little" else "big"
    if kind == "binary_output":
        code = data.get("code", "latch")
        if code not in CROB_CODES:
            raise ValueError("A CROB is latch, pulse or trip_close")
        command["code"] = code
    return jsonify(master().command(command))


@blueprint.put("/tools/master/ranges")
def save_ranges():
    """Modbus ranges to read; they may change while connected (the next poll uses them)."""
    current = master().config()
    ranges = validate_ranges(body().get("modbus_ranges"))
    master().save({**current, "modbus_ranges": ranges})
    return jsonify(modbus_ranges=ranges)


@blueprint.put("/tools/master/routes")
def save_routes():
    """Protocol converter: upstream command addresses relayed to device command addresses."""
    routes = validate_routes(body().get("routes"))
    master().save({**master().config(), "routes": routes})
    return jsonify(routes=routes)


@blueprint.post("/tools/master/probe")
def find_registers():
    """Find registers: reads only, bounded and paced (spec MASTER-07)."""
    data = body()
    return jsonify(master().find_registers(*validate_probe(data.get("table"), data.get("start"), data.get("end"))))


@blueprint.post("/tools/master/probe/ranges")
def add_found_ranges():
    """Add what the last Find registers found to the read list, as the chosen data type."""
    data_type = body().get("data_type", "uint16")
    found = master().probe.snapshot().get("found", [])
    current = master().config()
    ranges = validate_ranges(current["modbus_ranges"] + ranges_from_probe(found, data_type))
    master().save({**current, "modbus_ranges": ranges})
    return jsonify(modbus_ranges=ranges)


@blueprint.get("/tools/master/traffic")
def traffic():
    return jsonify(master().traffic.after(int(request.args.get("after", 0))))


@blueprint.get("/tools/master/commands")
def commands():
    return jsonify(master().commands.after(int(request.args.get("after", 0)), limit=200))
