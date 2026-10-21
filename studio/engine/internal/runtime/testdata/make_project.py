"""Write the synthetic runtime test project (public template IEDs only) as Python holds it.

Usage: studio/.venv/Scripts/python.exe make_project.py OUT.json [IP_PREFIX] [PORT_BASE]
The project is ProjectService.project after imports and edits, dumped with json.dumps, so the
Go runtime tests read exactly the dicts the Python coordinator indexes (keys, order, types).
"""
import json
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[5]
sys.path.insert(0, str(ROOT / "studio" / "backend"))

from studio_app.projects.service import ProjectService  # noqa: E402


def binding(ied, ref):
    return {"ied_id": ied["id"], "ref": ref, "report_ref": None}


def node(key, kind, label, **data):
    return {"id": key, "position": {"x": 0, "y": 0}, "data": {"label": label, "kind": kind, **data}}


def wire(key, source, target, source_handle=None, target_handle=None):
    return {"id": key, "source": source, "target": target,
            "sourceHandle": source_handle, "targetHandle": target_handle}


def build(directory, prefix, port_base):
    projects = ProjectService(Path(directory), lambda revision, source: None)
    bay = projects.add_template("BayA")[0]
    bay2 = projects.add_template("BayB")[0]
    bay.update(ip=f"{prefix}.11", port=port_base + 1)
    bay2.update(ip=f"{prefix}.12", port=port_base + 2)
    ctrl, ctrl2 = bay["name"] + "CTRL", bay2["name"] + "CTRL"
    nodes = [
        node("feed", "INFEED", "Grid", infeed_type="grid", infeed_state="live"),
        node("ds1", "DS", "Line DS", state_mapping={"mode": "direct", "source": binding(bay, ctrl + "/XSWI1.Pos.stVal")},
             control_binding={"ied_id": bay["id"], "ref": ctrl + "/XSWI1.Pos"}, initial_position="closed"),
        node("cb1", "CB", "Breaker", state_mapping={"mode": "direct", "source": binding(bay, ctrl + "/XCBR1.Pos.stVal")},
             control_binding={"ied_id": bay["id"], "ref": ctrl + "/XCBR1.Pos"}),
        node("es1", "ES", "Earth", state_mapping={"mode": "direct", "source": binding(bay, ctrl + "/XSWI2.Pos.stVal")},
             control_binding={"ied_id": bay["id"], "ref": ctrl + "/XSWI2.Pos"}),
        node("j1", "JUNCTION", "Bus"),
        node("cb2", "CB", "Feeder", state_mapping={"mode": "direct", "source": binding(bay2, ctrl2 + "/XCBR1.Pos.stVal")},
             control_binding={"ied_id": bay2["id"], "ref": ctrl2 + "/XCBR1.Pos"}),
        node("pb1", "PUSH_BUTTON", "Latch", push_mode="latching_status",
             state_mapping={"mode": "direct", "source": binding(bay2, ctrl2 + "/GGIO1.SPCSO1.stVal")}),
        node("trip", "TRIP_RELAY", "Trip", control_binding={"ied_id": bay["id"], "ref": ctrl + "/GGIO1.SPCSO1"},
             state_mapping={"mode": "direct", "source": binding(bay, ctrl + "/GGIO1.SPCSO1.stVal")}),
        node("relay", "RELAY", "Relay", relay_statuses=[
            {"id": "r1", "label": "Loc", "source": binding(bay, ctrl + "/XCBR1.Loc.stVal")}]),
    ]
    edges = [wire("w1", "feed", "ds1", "east", "north"), wire("w2", "ds1", "cb1", "south", "north"),
             wire("w3", "cb1", "j1", "south", None), wire("w4", "j1", "es1", None, "north"),
             wire("w5", "j1", "cb2", None, "north")]
    projects.diagram({"nodes": nodes, "edges": edges})
    projects.source(bay["id"], ctrl + "/GGIO1.AnIn1.mag.f", {"mode": "ramp", "offset": 1.5, "amplitude": 10,
                                                             "period": 2, "seed": 1, "overridden": False})
    projects.source(bay2["id"], ctrl2 + "/GGIO1.AnIn1.mag.f", {"mode": "noise", "offset": 0, "amplitude": 5,
                                                               "period": 10, "seed": 7, "overridden": False})
    projects.source(bay2["id"], ctrl2 + "/GGIO1.AnIn2.mag.f", {"mode": "sine", "offset": 0, "amplitude": 3,
                                                               "period": 1, "seed": 1, "overridden": False})
    projects.scenarios([{"id": "s1", "name": "Trip test", "enabled": False, "steps": [
        {"id": "a", "at": 0.2, "ied_id": bay["id"], "ref": ctrl + "/GGIO1.AnIn2.mag.f", "action": "set", "value": 42.5},
        {"id": "b", "at": 0.4, "ied_id": bay["id"], "ref": ctrl + "/GGIO1.AnIn2.mag.f", "action": "assert", "value": 42.5},
        {"id": "c", "at": 0.5, "ied_id": bay["id"], "ref": ctrl + "/XCBR1.Pos.stVal", "action": "assert", "value": 3}]}])
    return projects.project


def main():
    out = Path(sys.argv[1])
    prefix = sys.argv[2] if len(sys.argv) > 2 else "127.63.1"
    port_base = int(sys.argv[3]) if len(sys.argv) > 3 else 21000
    with tempfile.TemporaryDirectory() as directory:
        project = build(directory, prefix, port_base)
    out.write_text(json.dumps(project), encoding="utf-8")


if __name__ == "__main__":
    main()
