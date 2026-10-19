"""S1 idle run, S3 scale and S9 project operations."""
import json
import time

from engine_bench import project as bench_project
from engine_bench.procstats import Sampler


def _workers(engine):
    return [pid for pid, name in engine.children().items() if name.lower() == "studio_ied.exe"]


def _run_and_stop(engine, seconds):
    """Run, hold idle for `seconds` while sampling, stop; times and resources."""
    started = time.perf_counter()
    engine.json("POST", "/api/v1/runtime/start")
    bench_project.wait_state(engine, "running")
    run_seconds = time.perf_counter() - started
    sampler = Sampler(engine.job, engine.backend_names).start()
    time.sleep(seconds)
    resources = sampler.stop()
    workers = len(_workers(engine))
    started = time.perf_counter()
    engine.json("POST", "/api/v1/runtime/stop")
    while _workers(engine) and time.perf_counter() - started < 60:
        time.sleep(0.02)
    return {"run_seconds": round(run_seconds, 3), "workers": workers,
            "stop_seconds": round(time.perf_counter() - started, 3), "idle": resources}


def s1_idle(make_engine, ieds=1, points=5000, seconds=60):
    engine = make_engine()
    try:
        imports = bench_project.import_ieds(engine, ieds, points)
        bench_project.relocate(engine)
        idle_before = Sampler(engine.job, engine.backend_names)
        idle_before.start()
        time.sleep(min(5, seconds))
        after_import = idle_before.stop()
        result = _run_and_stop(engine, seconds)
        return {"scenario": "S1", "ieds": ieds, "points_per_ied": points, "seconds": seconds,
                "ready_seconds": round(engine.ready_seconds, 3), "import_seconds": [round(t, 3) for t in imports],
                "idle_after_import": after_import, **result}
    finally:
        engine.stop()


def s3_scale(make_engine, ieds=6, points=15000, seconds=20):
    engine = make_engine()
    try:
        imports = bench_project.import_ieds(engine, ieds, points)
        open_seconds = bench_project.relocate(engine)
        return {"scenario": "S3", "ieds": ieds, "points_per_ied": points, "import_seconds": sum(imports),
                "open_seconds": round(open_seconds, 3), **_run_and_stop(engine, seconds)}
    finally:
        engine.stop()


def _timed(action):
    started = time.perf_counter()
    action()
    return round(time.perf_counter() - started, 4)


def s9_project_ops(make_engine, ieds=2, points=2000, repeats=5):
    """Open, save, download, diagram edit and IED delete on a synthetic project."""
    engine = make_engine()
    try:
        bench_project.import_ieds(engine, ieds, points)
        open_seconds = bench_project.relocate(engine)
        timings = {"save": [], "download": [], "diagram_edit": [], "project_get": []}
        for repeat in range(repeats):
            timings["save"].append(_timed(lambda: engine.json("POST", "/api/v1/project/save")))
            timings["download"].append(_timed(lambda: engine.request("GET", "/api/v1/project/download")))
            timings["project_get"].append(_timed(lambda: bench_project.project(engine)))
            revision = bench_project.project(engine)["revision"]
            diagram = {"nodes": [{"id": f"cb{repeat}", "position": {"x": repeat * 40, "y": 0},
                                  "data": {"kind": "CB", "label": f"CB {repeat}"}}], "edges": []}
            timings["diagram_edit"].append(_timed(lambda: engine.json("PUT", "/api/v1/project/diagram", {
                "diagram": diagram, "revision": revision, "tab_id": "bench"})))
        victim = bench_project.project(engine)["ieds"][-1]["id"]
        delete = _timed(lambda: engine.json("DELETE", f"/api/v1/ieds/{victim}", {"confirm": True}))
        return {"scenario": "S9", "ieds": ieds, "points_per_ied": points, "open_seconds": round(open_seconds, 3),
                "timings": timings, "delete_ied_seconds": delete,
                "median": {key: sorted(values)[len(values) // 2] for key, values in timings.items()}}
    finally:
        engine.stop()


def dumps(result):
    return json.dumps(result, indent=2)
