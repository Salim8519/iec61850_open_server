"""CYCLES (repeated Run/Stop in one app) and APPCYCLES (repeated app start/shutdown on one data folder).

After every stop: native workers gone, IED ports refuse connections, backend memory/handles.
App cycles reuse one data folder (the single-instance lease must be released every time) and
alternate the two endings of shutdown.py: Ctrl+C (= closing the app window) and a backend kill.
"""
from pathlib import Path
import shutil
import tempfile
import time

from engine_bench import canvas, project as bench_project, shutdown
from engine_bench.machine import free_mb, listening
from engine_bench.procstats import sample


def _ieds(engine):
    return [(ied["ip"], ied["port"]) for ied in bench_project.project(engine)["ieds"]]


def _workers(engine):
    return [pid for pid, name in engine.children().items() if name.lower() == "studio_ied.exe"]


def _backend(engine):
    stats = [sample(pid) for pid in shutdown.backend_pids(engine)]
    stats = [item for item in stats if item]
    return {"ws_mb": round(sum(item["ws"] for item in stats) / 2**20, 1),
            "private_mb": round(sum(item["private"] for item in stats) / 2**20, 1),
            "handles": sum(item["handles"] for item in stats)}


def run_cycles(make_engine, ieds=10, points=2000, cycles=50, hold=2):
    engine = make_engine()
    try:
        bench_project.import_ieds(engine, ieds, points)
        bench_project.relocate(engine)
        addresses = _ieds(engine)
        baseline, rows = _backend(engine), []
        for cycle in range(cycles):
            row = {"cycle": cycle}
            try:
                started = time.perf_counter()
                engine.json("POST", "/api/v1/runtime/start")
                bench_project.wait_state(engine, "running", timeout=300)
                row["run_s"] = round(time.perf_counter() - started, 3)
                row["workers"] = len(_workers(engine))
                time.sleep(hold)
                started = time.perf_counter()
                engine.json("POST", "/api/v1/runtime/stop", timeout=300)
                row["stop_api_s"] = round(time.perf_counter() - started, 3)
                row["stop_s"] = shutdown.wait_gone(engine, _workers, 60)
                row["stop_s"] = None if row["stop_s"] is None else round(row["stop_api_s"] + row["stop_s"], 3)
                row["ports_still_open"] = sum(listening(ip, port) for ip, port in addresses)
                row["natives_left"] = sorted(set(shutdown.native_pids(engine).values()))
                row.update(_backend(engine))
            except Exception as error:  # record and continue: a failed cycle is the finding
                row["error"] = f"{type(error).__name__}: {error}"[:300]
            rows.append(row)
        good = [row for row in rows if "error" not in row]
        return {"scenario": "CYCLES", "ieds": ieds, "points_per_ied": points, "cycles": cycles,
                "baseline_backend": baseline, "rows": rows, "errors": len(rows) - len(good),
                "run_s_max": max((row["run_s"] for row in good), default=None),
                "run_s_mean": round(sum(row["run_s"] for row in good) / len(good), 3) if good else None,
                "stop_s_max": max((row["stop_s"] or 999 for row in good), default=None),
                "stop_s_mean": round(sum(row["stop_s"] or 999 for row in good) / len(good), 3) if good else None,
                "cycles_with_open_ports": sum(1 for row in good if row["ports_still_open"]),
                "cycles_with_natives_left": [row["cycle"] for row in good if row["natives_left"]]}
    finally:
        engine.stop()


def _archive(make_engine, ieds, points):
    engine = make_engine()
    try:
        bench_project.import_ieds(engine, ieds, points)
        bench_project.relocate(engine)
        canvas.add_breakers(engine, bench_project.project(engine)["ieds"])
        status, data, _ = engine.request("GET", "/api/v1/project/download")
        if status != 200:
            raise RuntimeError(f"download failed: {status}")
        return data
    finally:
        engine.stop()


def app_cycles(make_engine, new_engine, ieds=10, points=2000, cycles=10, hold=5):
    archive = _archive(make_engine, ieds, points)
    shared = Path(tempfile.mkdtemp(prefix="studio-appcycles-")) / "data"
    rows = []
    try:
        for cycle in range(cycles):
            mode = "ctrl_c" if cycle % 2 == 0 else "kill"
            row, engine = {"cycle": cycle, "mode": mode, "free_mb_before": free_mb()}, new_engine()
            engine.data, engine.popen_options = shared, shutdown.hidden_console()
            try:
                engine.start()
                row["ready_s"] = round(engine.ready_seconds, 3)
                started = time.perf_counter()
                engine.upload("/api/v1/project/open", "file", "cycles.subsim", archive)
                row["open_s"] = round(time.perf_counter() - started, 3)
                started = time.perf_counter()
                engine.json("POST", "/api/v1/runtime/start")
                bench_project.wait_state(engine, "running", timeout=300)
                row["run_s"] = round(time.perf_counter() - started, 3)
                addresses, row["workers"] = _ieds(engine), len(_workers(engine))
                row["operate_s"], row["operate_error"] = canvas.operate(engine, canvas.node_id(0), "close")
                time.sleep(hold)
                started = time.perf_counter()
                if mode == "ctrl_c":
                    row["signal_sent"] = shutdown.send_ctrl_c(engine.process.pid)
                else:
                    shutdown.kill_backend(engine)
                row["backend_exit_s"] = shutdown.wait_gone(engine, shutdown.backend_pids, 60)
                gone = shutdown.wait_gone(engine, shutdown.native_pids, 30)
                row["natives_exit_s"] = None if gone is None else round(time.perf_counter() - started, 3)
                row["natives_left"] = sorted(set(shutdown.native_pids(engine).values()))
                row["ports_still_open"] = sum(listening(ip, port) for ip, port in addresses)
                row["log_tail"] = [line.rstrip() for line in engine.output(3)]
            except Exception as error:
                row["error"] = f"{type(error).__name__}: {error}"[:300]
                row["log_tail"] = [line.rstrip() for line in engine.output(8)]
            finally:
                engine.stop()
            time.sleep(3)
            row["free_mb_after"] = free_mb()
            rows.append(row)
        good = [row for row in rows if "error" not in row]
        return {"scenario": "APPCYCLES", "ieds": ieds, "points_per_ied": points, "cycles": cycles,
                "errors": len(rows) - len(good), "rows": rows,
                "cycles_with_natives_left": [row["cycle"] for row in good if row["natives_left"]],
                "cycles_with_open_ports": [row["cycle"] for row in good if row["ports_still_open"]]}
    finally:
        shutil.rmtree(shared.parent, ignore_errors=True)
