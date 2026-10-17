"""Propose a gateway point list from the IEDs of the project (spec gateway.md).

Only useful values are proposed (gateway_selection.py): positions become double
points, status alarms and protection trips single points, measurement magnitudes
float (or integer) measurements, device health integers, and controllable objects
of simulated and real IEDs commands. Addresses follow one simple plan per protocol so a SCADA engineer can
read the list; every address stays editable afterwards.
"""
from .gateway_selection import dataset_refs, engineered, rank, smart_kind

PROTOCOLS = ("iec104", "modbus", "dnp3")
MONITOR_KINDS = ("single", "double", "float", "integer")
COMMAND_KINDS = ("command_single", "command_double")
KINDS = MONITOR_KINDS + COMMAND_KINDS
MAX_POINTS = 5000
# First address of each kind. IEC 104: IOA blocks; Modbus: discrete inputs (single),
# input/holding registers (double, float, integer) and coils (commands); DNP3 indexes.
START = {
    "iec104": {"single": 10001, "double": 20001, "float": 30001, "integer": 40001, "command": 50001},
    "modbus": {"single": 0, "double": 0, "float": 1000, "integer": 5000, "command": 0},
    "dnp3": {"single": 0, "double": 0, "float": 0, "integer": 0, "command": 0},
}
WIDTH = {"float": 2, "integer": 2}  # Modbus registers per value


def point_id(ied_id, ref, kind):
    return f"{ied_id}:{ref}:{kind}"


def _space(protocol, kind):
    """Address space a kind lives in; points sharing a space must not overlap."""
    if protocol == "iec104":
        return "ioa"
    if kind in COMMAND_KINDS:
        return "command"
    if protocol == "modbus":
        return "discrete" if kind == "single" else "register"
    return "analog" if kind in ("float", "integer") else kind


def _width(protocol, kind):
    return WIDTH.get(kind, 1) if protocol == "modbus" else 1


def occupied(protocol, point):
    """(space, address) cells used by one point."""
    space = _space(protocol, point["kind"])
    return [(space, point["address"] + offset) for offset in range(_width(protocol, point["kind"]))]


def propose(ieds, protocol):
    """Return (points, notes): the smart selection of every IED that is not switched off."""
    if protocol not in PROTOCOLS:
        raise ValueError("Choose IEC 104, Modbus TCP or DNP3")
    candidates = []
    for ied in ieds:
        if ied.get("role") == "off":
            continue
        refs = dataset_refs(ied)
        for signal in ied["signals"]:
            kind = smart_kind(signal)
            if kind and engineered(signal, refs):
                candidates.append({"kind": kind, "ied": ied["id"], "ref": signal["ref"]})
        for control in ied.get("controls", []):
            kind = "command_double" if control.get("cdc") == "DPC" else "command_single"
            candidates.append({"kind": kind, "ied": ied["id"], "ref": control["ref"]})
    notes = []
    candidates.sort(key=rank)  # stable: keeps the IED order inside a rank
    if len(candidates) > MAX_POINTS:
        notes.append(f"The IEDs offer {len(candidates)} points; the first {MAX_POINTS} were proposed. "
                     "Remove what SCADA does not need, or import your own point list as CSV.")
        candidates = candidates[:MAX_POINTS]
    used, points, cursor = set(), [], dict(START[protocol])
    order = {kind: index for index, kind in enumerate(KINDS)}
    for candidate in sorted(candidates, key=lambda item: order[item["kind"]]):
        block = "command" if candidate["kind"] in COMMAND_KINDS else candidate["kind"]
        point = {"id": point_id(candidate["ied"], candidate["ref"], candidate["kind"]), **candidate,
                 "address": cursor[block], "scale": 1.0, "description": ""}
        while any(cell in used for cell in occupied(protocol, point)):
            point["address"] += 1
        cells = occupied(protocol, point)
        used.update(cells)
        cursor[block] = cells[-1][1] + 1  # next free address of this block; never scans from the start again
        points.append(point)
    if not points:
        notes.append("No position, status, measurement or command was found. Import IEDs first, "
                     "or add points from a CSV point list.")
    return points, notes
