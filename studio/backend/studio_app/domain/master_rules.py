"""Validate the SCADA master settings (ScadaBench asking a gateway, RTU or outstation).

Words per protocol (ADR-0061): IEC 104 controlling station, DNP3 master, Modbus TCP client.
Every problem names the setting and the allowed values.
"""
import re

ROLES = ("gateway", "master", "converter")  # station gateway, SCADA master, protocol converter (ADR-0061)
PROTOCOLS = ("iec104", "dnp3", "modbus")
DEFAULT_SETTINGS = {
    "iec104": {"port": 2404, "common_address": 1, "originator_address": 0, "t0": 10, "t1": 15, "t2": 10,
               "t3": 20, "k": 12, "w": 8, "time_tagged_commands": False, "gi_minutes": 15},
    "dnp3": {"port": 20000, "outstation_address": 10, "master_address": 1, "integrity_minutes": 15,
             "event_seconds": 5, "unsolicited": True},
    "modbus": {"port": 502, "unit_id": 1, "poll_ms": 1000},
}
RANGES = {"port": (1, 65535), "common_address": (1, 65534), "originator_address": (0, 255), "t0": (1, 255),
          "t1": (1, 255), "t2": (1, 255), "t3": (1, 3600), "k": (1, 32767), "w": (1, 32767),
          "gi_minutes": (0, 1440), "outstation_address": (0, 65519), "master_address": (0, 65519),
          "integrity_minutes": (0, 1440), "event_seconds": (0, 3600), "unit_id": (0, 255), "poll_ms": (200, 60000)}
FLAGS = {"time_tagged_commands", "unsolicited"}
TABLES = {"coils": 2000, "discrete_inputs": 2000, "holding_registers": 125, "input_registers": 125}
DATA_TYPES = {"bool": 1, "uint16": 1, "int16": 1, "uint32": 2, "int32": 2, "float32": 2}
MAX_RANGES = 20
MAX_PROBE = 10000
MAX_LABELS = 20000
HOST = re.compile(r"^[A-Za-z0-9]([A-Za-z0-9.\-]{0,251}[A-Za-z0-9])?$")


def empty_master():
    return {"protocol": "iec104", "host": "", "auto_connect": False,
            "settings": {name: dict(values) for name, values in DEFAULT_SETTINGS.items()},
            "modbus_ranges": [], "routes": [], "unticked": {name: [] for name in PROTOCOLS},
            "labels": {name: {} for name in PROTOCOLS}}


def protocol_arg(value):
    if value not in PROTOCOLS:
        raise ValueError("Choose IEC 104 (iec104), DNP3 (dnp3) or Modbus TCP (modbus)")
    return value


def validate_host(value):
    host = value.strip() if isinstance(value, str) else ""
    if not HOST.match(host):
        raise ValueError("Type the device address, for example 192.168.1.20 or rtu1.station.local")
    return host


def validate_settings(protocol, values):
    result = dict(DEFAULT_SETTINGS[protocol_arg(protocol)])
    for key, value in (values or {}).items():
        if key not in result:
            raise ValueError(f"Unknown {protocol} setting '{key}'")
        if key in FLAGS:
            if not isinstance(value, bool):
                raise ValueError(f"{key.replace('_', ' ').capitalize()} is on or off")
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


def validate_ranges(ranges):
    """Modbus has no discovery: the user lists what to read. Each range is one request."""
    if not isinstance(ranges, list) or len(ranges) > MAX_RANGES:
        raise ValueError(f"Send at most {MAX_RANGES} Modbus ranges")
    clean = []
    for number, raw in enumerate(ranges, 1):
        if not isinstance(raw, dict) or raw.get("table") not in TABLES:
            raise ValueError(f"Range {number}: choose coils, discrete inputs, holding registers or input registers")
        bits = raw["table"] in ("coils", "discrete_inputs")
        data_type = raw.get("data_type", "bool" if bits else "uint16")
        if data_type not in DATA_TYPES or (data_type == "bool") != bits:
            raise ValueError(f"Range {number}: coils and inputs are read as bool, registers as uint16, int16, uint32, int32 or float32")
        start, count = raw.get("start"), raw.get("count")
        limit = TABLES[raw["table"]] // DATA_TYPES[data_type]
        if isinstance(start, bool) or not isinstance(start, int) or not 0 <= start <= 65535:
            raise ValueError(f"Range {number}: the start address is a whole number from 0 to 65535")
        if isinstance(count, bool) or not isinstance(count, int) or not 1 <= count <= limit:
            raise ValueError(f"Range {number}: read 1 to {limit} values of this type in one range")
        if start + count * DATA_TYPES[data_type] > 65536:
            raise ValueError(f"Range {number}: the range runs past address 65535")
        swap = raw.get("word_order", "big")
        if swap not in ("big", "little"):
            raise ValueError(f"Range {number}: the word order is big (high word first) or little")
        clean.append({"table": raw["table"], "start": start, "count": count, "data_type": data_type,
                      "word_order": swap})
    return clean


def validate_probe(table, start, end):
    """Find registers reads at most MAX_PROBE addresses of one table."""
    if table not in TABLES:
        raise ValueError("Choose coils, discrete inputs, holding registers or input registers")
    for value in (start, end):
        if isinstance(value, bool) or not isinstance(value, int) or not 0 <= value <= 65535:
            raise ValueError("Addresses are whole numbers from 0 to 65535")
    if end < start or end - start + 1 > MAX_PROBE:
        raise ValueError(f"Find registers checks 1 to {MAX_PROBE} addresses at a time")
    return table, start, end


def ranges_from_probe(found, data_type="uint16"):
    """Answering spans -> read ranges, split to the per-request limit of the table."""
    if data_type not in DATA_TYPES or data_type == "bool":
        raise ValueError("Read registers as uint16, int16, uint32, int32 or float32")
    ranges = []
    for span in found:
        bits = span["table"] in ("coils", "discrete_inputs")
        kind = "bool" if bits else data_type
        width = DATA_TYPES[kind]
        limit = TABLES[span["table"]] // width
        count = span["count"] // width
        for offset in range(0, count, limit):
            ranges.append({"table": span["table"], "start": span["start"] + offset * width,
                           "count": min(limit, count - offset), "data_type": kind, "word_order": "big"})
    return ranges


DEVICE_KINDS = {"single", "double", "binary_output", "coil"}


def validate_routes(routes):
    """User command routes: [{upstream, upstream_kind, kind, address, select}]."""
    if not isinstance(routes, list) or len(routes) > 500:
        raise ValueError("Send at most 500 command routes as a list")
    clean = []
    for number, raw in enumerate(routes, 1):
        if not isinstance(raw, dict) or raw.get("kind") not in DEVICE_KINDS:
            raise ValueError(f"Route {number}: the device command is single, double, binary_output or coil")
        if raw.get("upstream_kind") not in ("command_single", "command_double"):
            raise ValueError(f"Route {number}: the upstream command is a single or double command")
        for key in ("upstream", "address"):
            value = raw.get(key)
            if isinstance(value, bool) or not isinstance(value, int) or not 0 <= value <= 16777215:
                raise ValueError(f"Route {number}: {key} must be a whole number")
        clean.append({"upstream": raw["upstream"], "upstream_kind": raw["upstream_kind"], "kind": raw["kind"],
                      "address": raw["address"], "select": raw.get("select", True) is True})
    return clean


def _labels(raw):
    if not isinstance(raw, dict):
        return {}
    return {key: value[:80] for key, value in list(raw.items())[:MAX_LABELS]
            if isinstance(key, str) and len(key) <= 40 and isinstance(value, str) and value.strip()}


def normalize_master(raw):
    """Stored master block -> checked structure (missing parts get the defaults)."""
    result = empty_master()
    if raw is None:
        return result
    if not isinstance(raw, dict):
        raise ValueError("The master section must be an object")
    result["protocol"] = protocol_arg(raw.get("protocol", "iec104"))
    result["host"] = validate_host(raw["host"]) if raw.get("host") else ""
    result["auto_connect"] = raw.get("auto_connect") is True
    for name in PROTOCOLS:
        result["settings"][name] = validate_settings(name, (raw.get("settings") or {}).get(name))
        unticked = (raw.get("unticked") or {}).get(name, [])
        result["unticked"][name] = [key for key in unticked if isinstance(key, str)][:MAX_LABELS]
        result["labels"][name] = _labels((raw.get("labels") or {}).get(name))
    result["modbus_ranges"] = validate_ranges(raw.get("modbus_ranges", []))
    result["routes"] = validate_routes(raw.get("routes", []))
    return result


def validate_role(value):
    if value not in ROLES:
        raise ValueError("Choose Station gateway (gateway), SCADA master (master) or Protocol converter (converter)")
    return value
