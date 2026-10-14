"""Which parts of the single-line diagram are live, dead or earthed (display only).

Pure: the diagram plus authoritative switch positions in, section states out.
Nothing here reads or writes an IED. Spec: energization (POWER-01..06).
"""
SWITCHES = {"CB", "DS", "ES"}
# Items that join every wire attached to them into one electrical point.
JOINING = {"JUNCTION", "CT", "INFEED"}
# A VT ends a circuit: each of its wires stays separate, as if always open.
ENDS = {"VT"}
POWER = SWITCHES | JOINING | ENDS
CLOSED, OPEN = 2, 1
# Strongest first: the word shown when one item touches several sections.
ORDER = ("earth_fault", "live", "maybe_live", "earthed", "dead")
SIDE_A, SIDE_B = {"north", "west"}, {"south", "east"}


class _Sets:
    def __init__(self):
        self.parent = {}

    def find(self, item):
        self.parent.setdefault(item, item)
        while self.parent[item] != item:
            self.parent[item] = self.parent[self.parent[item]]
            item = self.parent[item]
        return item

    def join(self, a, b):
        self.parent[self.find(a)] = self.find(b)


def _terminal(node, handle, default):
    """The electrical point a wire end attaches to: the node, or one side of a switch/VT."""
    kind = node["data"]["kind"]
    if kind in JOINING:
        return node["id"]
    side = handle or default
    return f"{node['id']}:{'a' if side in SIDE_A else 'b'}"


def _position(positions, node_id):
    value = positions.get(node_id)
    return value if value in (OPEN, CLOSED) else None


class Sections:
    """Electrical sections of a diagram for one set of switch positions."""

    def __init__(self, diagram, positions):
        self.nodes = {node["id"]: node for node in diagram["nodes"]}
        self.positions = {node_id: _position(positions, node_id) for node_id, node in self.nodes.items()
                          if node["data"]["kind"] in SWITCHES}
        self.certain, self.possible = _Sets(), _Sets()
        self.wires = {}
        for edge in diagram["edges"]:
            source, target = self.nodes.get(edge["source"]), self.nodes.get(edge["target"])
            if not source or not target or not {source["data"]["kind"], target["data"]["kind"]} <= POWER:
                continue
            # React Flow attaches an unnamed source end to "east" and an unnamed target end to "north".
            ends = (_terminal(source, edge.get("sourceHandle"), "east"),
                    _terminal(target, edge.get("targetHandle"), "north"))
            self.wires[edge["id"]] = ends
            for sets in (self.certain, self.possible):
                sets.join(*ends)
        for node_id, position in self.positions.items():
            sides = (f"{node_id}:a", f"{node_id}:b")
            if self.nodes[node_id]["data"]["kind"] == "ES":
                continue
            if position == CLOSED:
                self.certain.join(*sides)
            if position != OPEN:
                self.possible.join(*sides)
        self.live = {self.certain.find(i) for i in self._feeds()}
        self.maybe = {self.possible.find(i) for i in self._feeds()}
        self.earthed = {self.certain.find(f"{node_id}:{side}") for node_id, position in self.positions.items()
                        if position == CLOSED and self.nodes[node_id]["data"]["kind"] == "ES" for side in "ab"}

    def _feeds(self):
        return [node_id for node_id, node in self.nodes.items()
                if node["data"]["kind"] == "INFEED" and node["data"].get("infeed_state", "live") != "dead"]

    def state(self, point):
        root = self.certain.find(point)
        live, earthed = root in self.live, root in self.earthed
        if live and earthed:
            return "earth_fault"
        if live:
            return "live"
        if earthed:
            return "earthed"
        return "maybe_live" if self.possible.find(point) in self.maybe else "dead"

    def points(self, node_id):
        """Electrical points of a power item: one, or its two sides."""
        kind = self.nodes[node_id]["data"]["kind"]
        return [node_id] if kind in JOINING else [f"{node_id}:a", f"{node_id}:b"]

    def position(self, node_id):
        return self.positions.get(node_id)


def energize(diagram, positions):
    """Return {"nodes": {id: [states strongest first]}, "edges": {id: state}} for power items."""
    return describe(Sections(diagram, positions))


def describe(sections):
    """Node and wire states of already-built sections."""
    edges = {edge_id: sections.state(ends[0]) for edge_id, ends in sections.wires.items()}
    wired = {point for ends in sections.wires.values() for point in ends}
    nodes = {}
    for node_id, node in sections.nodes.items():
        if node["data"]["kind"] not in POWER:
            continue
        # An unwired side of a switch says nothing; an unwired infeed still shows its own state.
        points = [point for point in sections.points(node_id) if point in wired]
        if not points and node["data"]["kind"] != "INFEED":
            continue
        states = {sections.state(point) for point in points or sections.points(node_id)}
        nodes[node_id] = [state for state in ORDER if state in states]
    return {"nodes": nodes, "edges": edges}
