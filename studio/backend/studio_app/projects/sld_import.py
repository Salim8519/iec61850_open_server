"""Preview a single-line diagram built from a stored SCL Substation section.

Preview never changes the project; the browser applies the returned diagram through
the normal revision-guarded diagram save, so it is one undo step.
"""
import copy
from studio_app.contracts.errors import StudioError
from studio_app.domain.sld_bindings import bind
from studio_app.domain.sld_layout import build_diagram
from studio_app.scl.substation import read_substation
from .scl_sources import require_source, stored_sources
from .validation import validate_diagram

BESIDE_GAP = 240


def substation_sources(service):
    """Every stored SCL file, saying whether it has a substation layout to draw."""
    with service.lock:
        result = []
        for entry in stored_sources(service.project).values():
            topology = read_substation(entry["text"])
            result.append({"id": entry["id"], "filename": entry["filename"], "ieds": entry["ieds"],
                           "substations": topology["substations"],
                           "equipment": len(topology["equipment"])})
        return result


def preview(service, source_id, mode):
    if mode not in {"replace", "beside"}:
        raise StudioError("Choose replace or beside")
    with service.lock:
        entry = require_source(service.project, source_id)
        topology = read_substation(entry["text"])
        if not topology["substations"]:
            raise StudioError("This file has no substation layout. Ask for the SCD from the system integrator.")
        try:
            built = build_diagram(topology, source_id)
        except ValueError as error:
            raise StudioError(str(error)) from error
        ieds = service.project["ieds"]
        drawn, unbound = _bind(built, ieds, topology["ld_domains"])
        current = copy.deepcopy(service.project["diagram"])
        diagram, present = _merge(current, built["diagram"], mode)
        diagram = validate_diagram(diagram, ieds)
        return {"diagram": diagram, "revision": service.revision, "layout": built["layout"],
                "drawn": drawn, "skipped": built["skipped"], "unbound": unbound,
                "diagnostics": built["diagnostics"], "already_present": present,
                "removed": len(current["nodes"]) if mode == "replace" else 0}


def _bind(built, ieds, domains):
    drawn, unbound = [], []
    for node in built["diagram"]["nodes"]:
        item = built["equipment"].get(node["id"])
        if item is None:
            continue
        kind = node["data"]["kind"]
        state, control, reasons = bind(kind, item["lnodes"], ieds, domains)
        if state:
            node["data"]["state_mapping"] = state
        if control:
            node["data"]["control_binding"] = control
        drawn.append({"path": item["path"], "kind": kind, "label": node["data"]["label"],
                      "bound": bool(state or control)})
        if reasons:
            unbound.append({"path": item["path"], "kind": kind, "reasons": reasons})
    return drawn, unbound


def _merge(current, built, mode):
    if mode == "replace":
        return built, []
    existing = {node["id"] for node in current["nodes"]}
    right = max((node["position"]["x"] for node in current["nodes"]), default=None)
    shift = 0 if right is None else right + BESIDE_GAP - min(
        (node["position"]["x"] for node in built["nodes"]), default=0)
    present, nodes = [], list(current["nodes"])
    for node in built["nodes"]:
        if node["id"] in existing:
            present.append(node["data"]["label"])
            continue
        node["position"] = {"x": node["position"]["x"] + shift, "y": node["position"]["y"]}
        nodes.append(node)
    wires = {edge["id"] for edge in current["edges"]}
    edges = list(current["edges"]) + [edge for edge in built["edges"] if edge["id"] not in wires]
    return {"nodes": nodes, "edges": edges}, present
