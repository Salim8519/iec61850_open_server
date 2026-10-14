"""Friendly switching hints from the drawing (spec interlock-hints, HINT-01..).

Advice only: nothing is ever blocked, and these are not the IED's interlocking (CILO).
Pure: built on the energisation sections of the current switch positions.
"""
from .energization import CLOSED, OPEN, Sections

PASS_THROUGH = {"JUNCTION", "CT"}
POWERED = {"live", "maybe_live", "earth_fault"}


def _neighbours(sections):
    """Electrical point -> points wired to it."""
    links = {}
    for a, b in sections.wires.values():
        links.setdefault(a, []).append(b)
        links.setdefault(b, []).append(a)
    return links


def _switches_near(sections, links, node_id):
    """Switches reached from either side of a switch through junctions and CTs only (its bay)."""
    found, seen = [], set(sections.points(node_id))
    queue = list(seen)
    while queue:
        for point in links.get(queue.pop(), []):
            if point in seen:
                continue
            seen.add(point)
            owner = point.split(":")[0]
            kind = sections.nodes[owner]["data"]["kind"]
            if kind in PASS_THROUGH:
                queue.append(point)
            elif kind in {"CB", "DS", "ES"} and owner != node_id and owner not in found:
                found.append(owner)
    return found


def _label(sections, node_id):
    return sections.nodes[node_id]["data"]["label"]


def _breaker_hints(sections, links, node_id):
    hints = []
    for other in _switches_near(sections, links, node_id):
        if sections.nodes[other]["data"]["kind"] != "CB":
            continue
        name, position = _label(sections, other), sections.position(other)
        if position == CLOSED:
            hints.append({"code": "load", "related": [other],
                          "text": f"Open {name} first: a disconnector must not switch load current."})
        elif position != OPEN:
            hints.append({"code": "unknown", "related": [other],
                          "text": f"The position of {name} is unknown. Check that it is open first."})
    return hints


def _closing_hints(sections, node_id):
    """Closing joins both sides: warn when that joins power to earth."""
    sides = [sections.state(point) for point in sections.points(node_id)]
    if sections.position(node_id) == CLOSED or len(set(sides)) < 2:
        return []
    if any(state in POWERED for state in sides) and "earthed" in sides:
        return [{"code": "earth", "related": [],
                 "text": "One side is earthed and the other side has power. Closing causes an earth fault. "
                         "Open the earth switch first."}]
    return []


def switching_hints(sections):
    """{node_id: {"open": [...], "close": [...]}} for switches that need a hint; others are left out."""
    links = _neighbours(sections)
    wired = set(links)
    result = {}
    for node_id, node in sections.nodes.items():
        kind = node["data"]["kind"]
        if kind not in {"CB", "DS", "ES"} or not wired & set(sections.points(node_id)):
            continue
        hints = {"open": [], "close": []}
        if kind == "DS":
            for action in hints:
                hints[action] = _breaker_hints(sections, links, node_id)
        if kind == "ES":
            states = {sections.state(point) for point in sections.points(node_id) if point in wired}
            if sections.position(node_id) != CLOSED and states & POWERED:
                hints["close"].append({"code": "earth", "related": [], "text":
                                       "This part may have power. Closing the earth switch causes an earth fault. "
                                       "Open the switches that feed it first."})
        else:
            hints["close"] += _closing_hints(sections, node_id)
        if hints["open"] or hints["close"]:
            result[node_id] = hints
    return result


def hints_for(diagram, positions):
    return switching_hints(Sections(diagram, positions))
