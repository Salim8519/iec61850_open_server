"""Deterministic grid placement for substation equipment without drawing coordinates.

Units are abstract grid cells (one equipment slot is 2 cells wide). Voltage levels
are rows, bays are columns, and equipment is stacked by its distance from a busbar.
"""
from collections import defaultdict

SLOT = 2.0
STEP = 2.5
BUS_STEP = 1.5
BAY_GAP = 1.0
LEVEL_GAP = 2.0


def bus_nodes(equipment, members, nodes):
    """Busbars are nodes in a bay without drawn equipment, or nodes spanning bays."""
    equipment_bays = {(item["level"], item["bay"]) for item in equipment}
    by_path = {item["path"]: item for item in equipment}
    result = []
    for path, member_paths in members.items():
        node = nodes[path]
        bays = {(by_path[p]["level"], by_path[p]["bay"]) for p in member_paths}
        if (node["level"], node["bay"]) not in equipment_bays or len(bays) > 1:
            result.append(path)
    return result


def _depths(equipment, members, buses):
    neighbours = defaultdict(list)
    for path, member_paths in members.items():
        for item in member_paths:
            neighbours[item].append(path)
    depth = {}
    frontier = sorted({p for bus in buses for p in members[bus]}, key=_order(equipment))
    seeds = list(frontier)
    for item in seeds:
        depth[item] = 0
    queue = list(seeds)
    order = _order(equipment)
    for start in sorted((item["path"] for item in equipment), key=order):
        if start not in depth:
            depth[start] = 0
            queue.append(start)
        while queue:
            current = queue.pop(0)
            for node in neighbours[current]:
                if node in buses:
                    continue
                for other in sorted(members[node], key=order):
                    if other not in depth:
                        depth[other] = depth[current] + 1
                        queue.append(other)
    return depth


def _order(equipment):
    index = {item["path"]: number for number, item in enumerate(equipment)}
    return lambda path: index[path]


def auto_positions(equipment, members, nodes):
    """Return ({equipment path: (x, y)}, {bus node path: y}) in grid cells."""
    buses = bus_nodes(equipment, members, nodes)
    depth = _depths(equipment, members, set(buses))
    positions, bus_y = {}, {}
    top = 0.0
    levels = sorted({item["level"] for item in equipment} | {nodes[b]["level"] for b in buses})
    for level in levels:
        level_buses = [b for b in buses if nodes[b]["level"] == level]
        for row, path in enumerate(level_buses):
            bus_y[path] = top + row * BUS_STEP + 0.5
        first = top + len(level_buses) * BUS_STEP + 1.0
        left, deepest = 0.0, 0
        bays = sorted({item["bay"] for item in equipment if item["level"] == level})
        for bay in bays:
            groups = defaultdict(list)
            for item in equipment:
                if (item["level"], item["bay"]) == (level, bay):
                    groups[depth[item["path"]]].append(item["path"])
            for level_depth, paths in groups.items():
                for slot, path in enumerate(paths):
                    positions[path] = (left + slot * SLOT, first + level_depth * STEP)
                deepest = max(deepest, level_depth)
            left += max(len(paths) for paths in groups.values()) * SLOT + BAY_GAP
        top = first + (deepest + 1) * STEP + LEVEL_GAP
    return positions, bus_y
