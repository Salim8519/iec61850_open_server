"""SCALE: per IED count, time to open/Run/Stop and steady RAM/CPU per process type, stopped and running.

Each level starts a fresh launcher, imports the synthetic IEDs, opens the relocated project,
samples `idle_seconds` stopped, runs, waits `settle` seconds, samples `seconds` running,
stops, and samples again stopped. A level is skipped when free RAM is under `min_free_mb`.
"""
import time

from engine_bench import project as bench_project
from engine_bench.machine import free_mb, listening
from engine_bench.pertype import by_type
from engine_bench.procstats import Sampler


def _sample(engine, seconds):
    sampler = Sampler(engine.job, engine.backend_names).start()
    time.sleep(seconds)
    rows = list(sampler.rows)
    return {"groups": sampler.stop(), "per_type": by_type(rows)}


def _workers(engine):
    return [pid for pid, name in engine.children().items() if name.lower() == "studio_ied.exe"]


def level(make_engine, ieds, points, seconds, idle_seconds=15, settle=10):
    free_before = free_mb()
    engine = make_engine()
    try:
        imports = bench_project.import_ieds(engine, ieds, points)
        open_seconds = bench_project.relocate(engine)
        stopped = _sample(engine, idle_seconds)
        started = time.perf_counter()
        engine.json("POST", "/api/v1/runtime/start")
        bench_project.wait_state(engine, "running", timeout=300)
        run_seconds = time.perf_counter() - started
        listening_ieds = sum(listening(ied["ip"], ied["port"]) for ied in bench_project.project(engine)["ieds"])
        time.sleep(settle)
        running = _sample(engine, seconds)
        free_running = free_mb()
        workers = len(_workers(engine))
        started = time.perf_counter()
        engine.json("POST", "/api/v1/runtime/stop", timeout=300)
        bench_project.wait_state(engine, "stopped", timeout=300)
        stop_api_seconds = time.perf_counter() - started
        while _workers(engine) and time.perf_counter() - started < 60:
            time.sleep(0.02)
        stop_seconds = time.perf_counter() - started
        after_stop = _sample(engine, idle_seconds)
        return {"scenario": "SCALE", "ieds": ieds, "points_per_ied": points, "ready_seconds": round(engine.ready_seconds, 3),
                "import_seconds_total": round(sum(imports), 3), "open_seconds": round(open_seconds, 3),
                "run_seconds": round(run_seconds, 3), "workers": workers, "listening_ieds": listening_ieds,
                "stop_api_seconds": round(stop_api_seconds, 3), "stop_until_no_worker_seconds": round(stop_seconds, 3),
                "workers_left_after_stop": len(_workers(engine)), "free_mb_before": free_before,
                "free_mb_running": free_running, "stopped_before_run": stopped, "running": running,
                "stopped_after_run": after_stop}
    finally:
        engine.stop()


def scale(make_engine, levels=(1, 5, 10, 20), points=2000, seconds=60, min_free_mb=2048):
    results = []
    for count in levels:
        free = free_mb()
        if free < min_free_mb + 40 * count:
            results.append({"scenario": "SCALE", "ieds": count, "skipped": f"free RAM {free} MB too low"})
            continue
        results.append(level(make_engine, count, points, seconds))
        time.sleep(5)
    return results
