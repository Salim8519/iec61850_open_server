"""Protocol converter: the device's points (read by ScadaBench as master) served upstream.

Every watched (ticked) point of the device becomes one upstream point with the same
address when that address is free in the upstream protocol, otherwise the next free one
(the move is listed in the notes). Upstream commands are relayed to the device: DNP3
binary outputs and Modbus coils automatically, any other command through a command route
the user adds (IEC 104 command addresses cannot be discovered).
"""
from .gateway_points import occupied
from .gateway_rules import ADDRESS_RANGE

PROTOCOL_NAMES = {"iec104": "IEC 104", "dnp3": "DNP3", "modbus": "Modbus"}

KIND_FOR = {
    "single": "single", "binary": "single", "coil": "single", "discrete_input": "single", "binary_output": "single",
    "double": "double", "double_binary": "double",
    "step": "integer", "bitstring": "integer", "counter": "integer", "frozen_counter": "integer",
    "normalized": "float", "scaled": "float", "float": "float", "analog": "float", "analog_output": "float",
    "holding_register": "float", "input_register": "float",
}
AUTO_COMMANDS = {"binary_output", "coil"}  # the device's command index equals its status index


def _kind(row):
    kind = KIND_FOR[row["group"]]
    if row["group"] in ("holding_register", "input_register") and row.get("type") in ("uint16", "int16", "uint32", "int32"):
        return "integer"
    return kind


def _place(protocol, point, cells):
    """Same address when free, else the next free one; None when the protocol has no room."""
    low, high = ADDRESS_RANGE[protocol]
    address = max(point["address"], low)
    while address <= high:
        candidate = dict(point, address=address)
        if not any(cell in cells for cell in occupied(protocol, candidate)):
            cells.update(occupied(protocol, candidate))
            return candidate
        address += 1
    return None


def _why_moved(protocol, address):
    """Change 0089: IEC 104 has no object address 0, which is not "taken"."""
    low = ADDRESS_RANGE[protocol][0]
    return f"{PROTOCOL_NAMES[protocol]} has no address {address}" if address < low else f"{address} was taken"


def map_points(rows, protocol, routes=()):
    """(upstream points, notes) for the watched device rows and the user's command routes."""
    cells, points, notes = set(), [], []
    for row in rows:
        if not row.get("ticked", True):
            continue
        name = row.get("label") or f"{row['group_name']} {row['address']}"
        point = _place(protocol, {"id": f"remote:{row['key']}", "kind": _kind(row), "address": row["address"],
                                  "scale": 1.0, "ied": "", "ref": row["key"], "description": name,
                                  "source": "remote", "remote": row["key"]}, cells)
        if point is None:
            notes.append(f"{name} is not served: no free address left in this protocol")
            continue
        if point["address"] != row["address"]:
            notes.append(f"{name} is served at address {point['address']} ({_why_moved(protocol, row['address'])})")
        points.append(point)
        if row["group"] in AUTO_COMMANDS:
            command = _place(protocol, {"id": f"remote-command:{row['key']}", "kind": "command_single",
                                        "address": row["address"], "scale": 1.0, "ied": "", "ref": row["key"],
                                        "description": f"Command {name}", "source": "remote",
                                        "route": {"kind": row["group"], "address": row["address"]}}, cells)
            if command is not None:
                points.append(command)
    for number, route in enumerate(routes, 1):
        command = _place(protocol, {"id": f"route:{number}", "kind": route["upstream_kind"], "address": route["upstream"],
                                    "scale": 1.0, "ied": "", "ref": f"route {number}",
                                    "description": f"Command to device {route['kind']} {route['address']}",
                                    "source": "remote", "route": {"kind": route["kind"], "address": route["address"],
                                                                  "select": route.get("select", True)}}, cells)
        if command is None or command["address"] != route["upstream"]:
            notes.append(f"Command route {number}: upstream address {route['upstream']} is already used")
            continue
        points.append(command)
    return points, notes


def device_value(route, value):
    """Upstream ON/OFF (close/open) -> the device command value."""
    if route["kind"] == "double":
        return 2 if value else 1
    return 1 if value else 0


def remote_sample(point, row, protocol_good, link_open):
    """The upstream value of a remote point, in the gateway's sample format."""
    if row is None or row.get("value") is None:
        return {"id": point["id"], "value": 0, "quality": 0, "time_ms": 0, "source_ok": False}
    value = row["value"]
    value = int(value) if point["kind"] in ("single", "double", "integer") else float(value)
    good = link_open and protocol_good
    return {"id": point["id"], "value": value, "quality": 0 if good else 2, "time_ms": row.get("time_ms") or 0,
            "source_ok": good}
