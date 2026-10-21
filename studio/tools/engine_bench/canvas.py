"""Single-line diagram breakers bound to the sample breaker of each IED, operated like the canvas.

The canvas sends POST /api/v1/equipment/<node>/operate {"action": "open"|"close"}
(DiagramPage.tsx), which runs the MMS command through studio_probe and applies the feedback.
"""
import time

CONTROL = "CTRL/XCBR1.Pos"
STATUS = CONTROL + ".stVal"


def node_id(index):
    return f"benchcb{index:02d}"


def add_breakers(engine, ieds):
    """One CB per IED, state from and command to its XCBR1.Pos (written while stopped)."""
    nodes = [{"id": node_id(index), "position": {"x": 120 * index, "y": 0},
              "data": {"label": f"CB {ied['name']}", "kind": "CB",
                       "state_mapping": {"mode": "direct", "source": {
                           "ied_id": ied["id"], "ref": ied["name"] + STATUS, "report_ref": None}},
                       "control_binding": {"ied_id": ied["id"], "ref": ied["name"] + CONTROL}}}
             for index, ied in enumerate(ieds)]
    revision = engine.json("GET", "/api/v1/project")["revision"]
    engine.json("PUT", "/api/v1/project/diagram", {"diagram": {"nodes": nodes, "edges": []},
                                                   "revision": revision, "tab_id": "bench"})
    return [node["id"] for node in nodes]


def operate(engine, node, action):
    """One canvas command: (seconds, error text or None)."""
    started = time.perf_counter()
    status, payload, _ = engine.request("POST", f"/api/v1/equipment/{node}/operate", {"action": action}, timeout=60)
    elapsed = time.perf_counter() - started
    return elapsed, None if status == 200 else f"{status}: {payload[:200].decode(errors='replace')}"
