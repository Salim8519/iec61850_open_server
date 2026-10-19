"""Cross-engine .subsim check run by service_roundtrip_test.go (needs STUDIO_PYTHON).

  python roundtrip.py make <out.subsim>   build a project with the frozen Python service and
                                          export it; prints the snapshot
  python roundtrip.py open <in.subsim>    restore an archive (persist=False); prints the
                                          snapshot
  python roundtrip.py time <in.subsim>    median restore time over 5 runs (seconds)
  python roundtrip.py big <out.subsim>    the timing project: template plus every IED of the
                                          public SCDs (scd/*.scd, demo station) up to 20
  python roundtrip.py savetime <in.subsim>  "<median restore> <first save> <median save>"

Snapshots are json.dumps(snapshot) without session_id and revision.
"""
import json
import sys
import tempfile
import time
from pathlib import Path

HERE = Path(__file__).resolve().parent
WORKTREE = HERE.parents[5]
sys.path.insert(0, str(WORKTREE / "studio" / "backend"))

from studio_app.projects.service import ProjectService  # noqa: E402
from studio_app.scl.document import preview_ieds  # noqa: E402


def diagram_for(service):
    ied = service.project["ieds"][0]
    source = {"ied_id": ied["id"], "ref": "BayControllerCTRL/XCBR1.Pos.stVal", "report_ref": None}
    node = {"id": "CB1", "type": "equipment", "position": {"x": 100, "y": 40.5},
            "data": {"label": "CB 1", "kind": "CB", "state_mapping": {"mode": "direct", "source": source},
                     "control_binding": {"ied_id": ied["id"], "ref": "BayControllerCTRL/XCBR1.Pos"}}}
    junction = {"id": "J1", "position": {"x": 0, "y": 0}, "data": {"label": "J", "kind": "JUNCTION"}}
    return {"nodes": [node, junction], "edges": [{"id": "w1", "source": "CB1", "target": "J1"}]}


def snapshot(service):
    result = service.snapshot()
    result.pop("session_id")
    result.pop("revision")
    return json.dumps(result)


def make_big(service):
    service.add_template("BayController")
    sources = sorted((WORKTREE / "scd").glob("*.scd"))
    sources.append(WORKTREE / "marketing" / "videos" / "demo" / "scadabench_demo.scd")
    for source in sources:
        data = source.read_bytes()
        taken = {ied["name"] for ied in service.project["ieds"]}
        names = [ied["name"] for ied in preview_ieds(data) if ied["name"] not in taken]
        names = names[:20 - len(service.project["ieds"])]
        if names:
            service.add_scl(data, source.name, names)


def save_times(service, payload):
    restores, saves = [], []
    for _ in range(5):
        started = time.perf_counter()
        service.restore(payload, persist=False)
        restores.append(time.perf_counter() - started)
    for _ in range(5):
        started = time.perf_counter()
        service.save()
        saves.append(time.perf_counter() - started)
    return sorted(restores)[2], saves[0], sorted(saves)[2]


def main():
    mode, path = sys.argv[1], Path(sys.argv[2])
    with tempfile.TemporaryDirectory() as folder:
        service = ProjectService(Path(folder))
        if mode == "make":
            service.add_template("BayController")
            model = (WORKTREE / "scd" / "model_substation.scd").read_bytes()
            service.add_scl(model, "model_substation.scd", ["FEED1", "BUS1"])
            service.diagram(diagram_for(service))
            path.write_bytes(service.export())
            print(snapshot(service))
        elif mode == "big":
            make_big(service)
            path.write_bytes(service.export())
        elif mode == "savetime":
            print(*save_times(service, path.read_bytes()))
        elif mode == "open":
            service.restore(path.read_bytes(), persist=False)
            print(snapshot(service))
        else:
            payload, runs = path.read_bytes(), []
            for _ in range(5):
                started = time.perf_counter()
                service.restore(payload, persist=False)
                runs.append(time.perf_counter() - started)
            print(sorted(runs)[2])


main()
