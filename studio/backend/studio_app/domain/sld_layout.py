"""Turn a parsed SCL substation topology into a ScadaBench single-line diagram.

Pure and deterministic: the same topology and source id always give the same node
ids, positions and wires. Bindings are applied by the caller.
"""
import hashlib
from collections import defaultdict
from .sld_auto_layout import auto_positions

GROUNDED = "grounded"
CARD_W, CARD_H, DOT = 108, 130, 40
# Space kept around cards so a junction and its wires fit between neighbours.
GAP_X, GAP_Y = 40, 80
GRID = 20
MIN_SCALE, MAX_SCALE = 90, 300
MAX_NODES, MAX_WIRES = 2000, 5000
NOT_DRAWN = "ScadaBench has no symbol for this equipment type yet"


def kind_of(item):
    kind = {"CBR": "CB", "DIS": "DS", "CTR": "CT", "VTR": "VT"}.get(item["type"])
    if kind == "DS" and GROUNDED in item["terminals"]:
        return "ES"
    return kind


def _id(source_id, text):
    return "scl-" + hashlib.sha256(f"{source_id}/{text}".encode()).hexdigest()[:16]


def _snap(value):
    return round(value / GRID) * GRID


def _scale(points):
    """Smallest scale at which no two equipment cards overlap, or None."""
    needed = MIN_SCALE
    items = sorted(points.values())
    for index, (ax, ay) in enumerate(items):
        for bx, by in items[index + 1:]:
            dx, dy = abs(ax - bx), abs(ay - by)
            fit = min((CARD_W + GAP_X) / dx if dx else float("inf"),
                      (CARD_H + GAP_Y) / dy if dy else float("inf"))
            needed = max(needed, fit)
    if needed > MAX_SCALE:
        return None
    return int(-(-needed // 10) * 10)


def _members(drawn):
    members = defaultdict(list)
    for item in drawn:
        for path in item["terminals"]:
            if path and path != GROUNDED and item["path"] not in members[path]:
                members[path].append(item["path"])
    return dict(members)


def _placement(drawn, members, nodes, diagnostics):
    esld = {item["path"]: (item["esld"]["x"], item["esld"]["y"]) for item in drawn if item["esld"]}
    if drawn and len(esld) == len(drawn):
        scale = _scale(esld)
        if scale:
            bus_y = {path: nodes[path]["bus_y"] for path in members if nodes[path]["bus_y"] is not None}
            return "esld", esld, bus_y, scale
        diagnostics.append("Drawing coordinates in the SCD overlap; the automatic layout was used")
    elif esld:
        diagnostics.append("Only some equipment has drawing coordinates; the automatic layout was used")
    grid, bus_y = auto_positions(drawn, members, nodes)
    return "automatic", grid, bus_y, MIN_SCALE


def _wire(source_id, a, b, centres):
    (ax, ay), (bx, by) = centres[a], centres[b]
    if abs(by - ay) >= abs(bx - ax):
        upper, lower = (a, b) if (ay, ax) <= (by, bx) else (b, a)
        ends = (upper, "south", lower, "north")
    else:
        left, right = (a, b) if ax <= bx else (b, a)
        ends = (left, "east", right, "west")
    first, second = sorted((a, b))
    return {"id": _id(source_id, f"wire|{first}|{second}"), "source": ends[0], "sourceHandle": ends[1],
            "target": ends[2], "targetHandle": ends[3]}


def build_diagram(topology, source_id):
    """Return {diagram, drawn, skipped, diagnostics, layout}; bindings are left empty."""
    diagnostics = list(topology["diagnostics"])
    drawn = [item for item in topology["equipment"] if kind_of(item)]
    skipped = [{"path": item["path"], "type": item["type"], "reason": NOT_DRAWN}
               for item in topology["equipment"] if not kind_of(item)]
    skipped += [{"path": item["path"], "type": item["element"], "reason": NOT_DRAWN}
                for item in topology["other"]]
    members = _members(drawn)
    nodes = topology["nodes"]
    layout, points, bus_y, scale = _placement(drawn, members, nodes, diagnostics)
    ids = {item["path"]: _id(source_id, item["path"]) for item in drawn}
    diagram_nodes, centres = [], {}
    for item in drawn:
        x, y = _snap(points[item["path"]][0] * scale), _snap(points[item["path"]][1] * scale)
        node_id = ids[item["path"]]
        centres[node_id] = (x + CARD_W / 2, y + CARD_H / 2)
        diagram_nodes.append({"id": node_id, "type": "equipment", "position": {"x": x, "y": y},
                              "data": {"label": (item["name"] or kind_of(item))[:100],
                                       "kind": kind_of(item), "rotation": 0}})
    edges = []
    for path, member_paths in members.items():
        owners = [ids[p] for p in member_paths]
        label = f"{nodes[path]['bay_name']}/{nodes[path]['name']}"[:100]
        if len(owners) == 2:
            edges.append(_wire(source_id, owners[0], owners[1], centres))
            continue
        if len(owners) == 1:
            junctions, slots = _single(path, owners[0], centres, members, ids)
        else:
            height = bus_y[path] * scale if path in bus_y else _gap_middle(owners, centres)
            slots = {owner: _row_x(centres[owner], height) for owner in owners}
            junctions = {x: (x, height) for x in sorted(set(slots.values()))}
        keys = {}
        for x, (cx, cy) in sorted(junctions.items()):
            node_id = _id(source_id, f"{path}#{x}")
            keys[x] = node_id
            centres[node_id] = (cx, cy)
            diagram_nodes.append({"id": node_id, "type": "equipment",
                                  "position": {"x": cx - DOT / 2, "y": cy - DOT / 2},
                                  "data": {"label": label, "kind": "JUNCTION", "rotation": 0}})
        chain = [keys[x] for x in sorted(keys)]
        edges.extend(_wire(source_id, a, b, centres) for a, b in zip(chain, chain[1:]))
        for owner in owners:
            edges.append(_wire(source_id, owner, keys[slots[owner]], centres))
    edges = list({edge["id"]: edge for edge in edges}.values())
    if len(diagram_nodes) > MAX_NODES or len(edges) > MAX_WIRES:
        raise ValueError(f"This substation needs {len(diagram_nodes)} symbols and {len(edges)} wires; "
                         f"the diagram limit is {MAX_NODES} symbols and {MAX_WIRES} wires")
    return {"diagram": {"nodes": diagram_nodes, "edges": edges}, "skipped": skipped,
            "equipment": {ids[item["path"]]: item for item in drawn},
            "diagnostics": diagnostics, "layout": layout}


def _single(path, owner, centres, members, ids):
    """Put a lone junction on the side away from the equipment's other connections."""
    cx, cy = centres[owner]
    others = [centres[ids[member]][1] for node, paths in members.items() if node != path
              and any(ids[member] == owner for member in paths)
              for member in paths if ids[member] != owner]
    below = bool(others) and sum(others) / len(others) > cy
    offset = -(CARD_H / 2 + 50) if below or not others else CARD_H / 2 + 50
    return {round(cx): (round(cx), cy + offset)}, {owner: round(cx)}


def _gap_middle(owners, centres):
    """A tee without a drawn bar sits in the widest vertical gap between its members."""
    heights = sorted(centres[owner][1] for owner in owners)
    gaps = [(b - a, (a + b) / 2) for a, b in zip(heights, heights[1:])]
    return max(gaps)[1] if gaps and max(gaps)[0] > 0 else heights[0] - CARD_H / 2 - 40


def _row_x(centre, height):
    """A card the bar runs through gets its junction beside it, not on top of it."""
    cx, cy = centre
    if abs(height - cy) < CARD_H / 2 + DOT / 2:
        return round(cx + CARD_W / 2 + 40)
    return round(cx)
