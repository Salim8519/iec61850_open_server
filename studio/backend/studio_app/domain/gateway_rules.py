"""Validate gateway settings and point lists; every problem names the point and the fix."""
import ipaddress
import math
from .gateway_points import COMMAND_KINDS, KINDS, MAX_POINTS, PROTOCOLS, occupied, point_id
from .master_rules import empty_master, normalize_master, validate_role

DEFAULT_SETTINGS = {
    "iec104": {"port": 2404, "common_address": 1, "t1": 15, "t2": 10, "t3": 20, "k": 12, "w": 8},
    "modbus": {"port": 502, "unit_id": 1},
    "dnp3": {"port": 20000, "outstation_address": 10, "master_address": 1, "unsolicited": True},
}
RANGES = {"port": (1, 65535), "common_address": (1, 65534), "t1": (1, 255), "t2": (1, 255), "t3": (1, 3600),
          "k": (1, 32767), "w": (1, 32767), "unit_id": (0, 255), "outstation_address": (0, 65519),
          "master_address": (0, 65519)}
ADDRESS_RANGE = {"iec104": (1, 16777215), "modbus": (0, 65535), "dnp3": (0, 65535)}
LISTEN = {"this_pc": "127.0.0.1", "network": "0.0.0.0"}


def empty_gateway():
    return {"role": "gateway", "master": empty_master(),
            "protocol": "iec104", "listen": "this_pc", "real_commands": True, "auto_start": False,
            "settings": {key: dict(value) for key, value in DEFAULT_SETTINGS.items()},
            "points": {protocol: [] for protocol in PROTOCOLS}}


def validate_settings(protocol, values):
    if protocol not in PROTOCOLS:
        raise ValueError("Choose IEC 104, Modbus TCP or DNP3")
    result = dict(DEFAULT_SETTINGS[protocol])
    for key, value in (values or {}).items():
        if key not in result:
            raise ValueError(f"Unknown {protocol} setting '{key}'")
        if key == "unsolicited":
            if not isinstance(value, bool):
                raise ValueError("Unsolicited reporting is on or off")
        else:
            low, high = RANGES[key]
            if isinstance(value, bool) or not isinstance(value, int) or not low <= value <= high:
                raise ValueError(f"{key.replace('_', ' ').capitalize()} must be a whole number from {low} to {high}")
        result[key] = value
    if protocol == "iec104" and not result["t2"] < result["t1"]:
        raise ValueError("t2 must be shorter than t1 (IEC 60870-5-104)")
    if protocol == "iec104" and not result["w"] <= result["k"]:
        raise ValueError("w must not be larger than k (IEC 60870-5-104)")
    return result


def _target(point, ieds_by_id, signals):
    ied = ieds_by_id.get(point.get("ied"))
    if ied is None:
        return None, "its IED is not in the project"
    if point["kind"] in COMMAND_KINDS:
        controls = {control["ref"]: control for control in ied.get("controls", [])}
        if point["ref"] not in controls:
            return ied, f"{ied['name']} has no controllable object {point['ref']}"
        return ied, ""
    if ied["id"] not in signals:  # one index per IED, built on first use: large SCDs stay fast
        signals[ied["id"]] = {item["ref"]: item for item in ied["signals"]}
    signal = signals[ied["id"]].get(point["ref"])
    if signal is None:
        return ied, f"{ied['name']} has no attribute {point['ref']}"
    if point["kind"] not in allowed_kinds(signal["kind"]):
        return ied, f"{point['ref']} is a {signal['kind']} value and cannot be sent as {point['kind']}"
    return ied, ""


def allowed_kinds(value_kind):
    """Gateway kinds that can carry an IEC 61850 value kind without changing its meaning."""
    if value_kind == "BOOLEAN":
        return {"single"}
    if value_kind == "Dbpos":
        return {"double"}
    if value_kind.startswith("FLOAT"):
        return {"float"}
    if value_kind.startswith("INT") or value_kind == "Enum":
        return {"integer", "float"}
    return set()


def validate_points(protocol, points, ieds):
    """Return (clean points, problems). Problems are sentences that name the point."""
    if not isinstance(points, list):
        raise ValueError("Send the point list as a list")
    if len(points) > MAX_POINTS:
        raise ValueError(f"A gateway holds at most {MAX_POINTS} points")
    ieds_by_id = {ied["id"]: ied for ied in ieds}
    low, high = ADDRESS_RANGE[protocol]
    clean, problems, cells, signals = [], [], {}, {}
    for number, raw in enumerate(points, 1):
        label = f"Point {number}"
        if not isinstance(raw, dict) or raw.get("kind") not in KINDS:
            problems.append(f"{label}: choose a type ({', '.join(KINDS)})")
            continue
        address, scale = raw.get("address"), raw.get("scale", 1.0)
        if isinstance(address, bool) or not isinstance(address, int) or not low <= address <= high:
            problems.append(f"{label}: the address must be a whole number from {low} to {high}")
            continue
        if isinstance(scale, bool) or not isinstance(scale, (int, float)) or not math.isfinite(scale) or scale == 0:
            problems.append(f"{label}: the scale must be a number other than 0")
            continue
        point = {"kind": raw["kind"], "ied": str(raw.get("ied", "")), "ref": str(raw.get("ref", "")),
                 "address": address, "scale": float(scale), "description": str(raw.get("description", ""))[:120]}
        ied, reason = _target(point, ieds_by_id, signals)
        label = f"{label} ({ied['name'] if ied else '?'} {point['ref']})"
        if reason:
            problems.append(f"{label}: {reason}")
            continue
        clash = next((cells[cell] for cell in occupied(protocol, point) if cell in cells), None)
        if clash:
            problems.append(f"{label}: address {address} is already used by point {clash}")
            continue
        for cell in occupied(protocol, point):
            cells[cell] = number
        point["id"] = point_id(point["ied"], point["ref"], point["kind"])
        clean.append(point)
    return clean, problems


def listen_address(choice):
    if choice not in LISTEN:
        raise ValueError("Choose 'This PC only' or 'All network adapters'")
    ipaddress.IPv4Address(LISTEN[choice])
    return LISTEN[choice]


def normalize_gateway(raw):
    """Stored gateway section -> checked structure. Point references are checked at
    Start against the IEDs of that moment, so a removed IED never deletes points."""
    if raw is None:
        return empty_gateway()
    if not isinstance(raw, dict):
        raise ValueError("The gateway section must be an object")
    result = empty_gateway()
    protocol = raw.get("protocol", "iec104")
    if protocol not in PROTOCOLS:
        raise ValueError(f"Unknown gateway protocol '{protocol}'")
    listen_address(raw.get("listen", "this_pc"))
    result.update(protocol=protocol, listen=raw.get("listen", "this_pc"), role=validate_role(raw.get("role", "gateway")),
                  master=normalize_master(raw.get("master")))
    for flag in ("real_commands", "auto_start"):
        if isinstance(raw.get(flag), bool):
            result[flag] = raw[flag]
    for name in PROTOCOLS:
        result["settings"][name] = validate_settings(name, (raw.get("settings") or {}).get(name))
        points = (raw.get("points") or {}).get(name, [])
        if not isinstance(points, list) or len(points) > MAX_POINTS:
            raise ValueError(f"The {name} point list must be a list of at most {MAX_POINTS} points")
        for point in points:
            if not isinstance(point, dict) or point.get("kind") not in KINDS or not isinstance(point.get("address"), int):
                raise ValueError(f"A stored {name} gateway point has no valid type or address")
            point.setdefault("id", point_id(point.get("ied", ""), point.get("ref", ""), point["kind"]))
        result["points"][name] = points
    return result
