"""SOAK: hours of running with generators, a canvas command every few seconds, SSE and /health.

Every `interval` seconds one row goes to the JSONL `stream` (so a long run can be watched and
survives an interrupted benchmark): per-process working set, private bytes, handles, threads
and CPU, data folder and launcher output size, free RAM, SSE delivery and lag, /health and
command latency, and worker PIDs that vanished or appeared (crash/restart). The run ends early
when free RAM drops under `min_free_mb`.
"""
import json
import os
from pathlib import Path
import time

from engine_bench import canvas, project as bench_project, soak_report
from engine_bench.machine import free_mb, thread_counts
from engine_bench.procstats import sample
from engine_bench.scenarios_load import _configure_sources
from engine_bench.soak_feed import Periodic, SoakSse, breaker_rounds, health_poll
from engine_bench.sse_clients import percentile


def _folder_mb(path):
    total = 0
    for root, _dirs, files in os.walk(path):
        for name in files:
            try:
                total += os.path.getsize(os.path.join(root, name))
            except OSError:
                pass
    return round(total / 2**20, 3)


def _processes(engine):
    threads, result = thread_counts(), {}
    for pid, name in engine.job.processes().items():
        stats = sample(pid)
        if stats:
            result[pid] = {"name": name.lower(), "ws": stats["ws"], "private": stats["private"],
                           "handles": stats["handles"], "cpu": round(stats["cpu"], 3), "threads": threads.get(pid)}
    return result


def _totals(processes, select):
    chosen = [item for item in processes.values() if select(item)]
    return (round(sum(item["ws"] for item in chosen) / 2**20, 2), round(sum(item["private"] for item in chosen) / 2**20, 2),
            sum(item["handles"] for item in chosen), sum(item["threads"] or 0 for item in chosen), len(chosen))


def _calls(results, origin):
    return [(round(at - origin, 2), seconds, error) for at, seconds, error in results]


def soak(make_engine, ieds=20, points=2000, sources=100, seconds=7200, interval=10, operate_every=5,
         health_every=2, stream=None, min_free_mb=2048, push_topics=()):
    engine = make_engine()
    feeds, rows, operates, health, ended = [], [], [], [], "completed"
    out = Path(stream).open("a", encoding="utf-8") if stream else None
    try:
        bench_project.import_ieds(engine, ieds, points)
        bench_project.relocate(engine)
        nodes = canvas.add_breakers(engine, bench_project.project(engine)["ieds"])
        configured = _configure_sources(engine, sources)
        engine.json("POST", "/api/v1/runtime/start")
        bench_project.wait_state(engine, "running", timeout=300)
        origin_wall, origin = time.time(), time.monotonic()
        sse = SoakSse(engine, push_topics).start()
        feeds = [sse, Periodic(health_every, health_poll(engine), "soak-health").start(),
                 Periodic(operate_every, breaker_rounds(engine, nodes), "soak-operate").start()]
        previous, cpu_before = None, {}
        while time.monotonic() - origin < seconds:
            time.sleep(max(0.0, interval - ((time.monotonic() - origin) % interval)))
            processes = _processes(engine)
            kinds, lags = sse.drain()
            health_calls, operate_calls = feeds[1].drain(), feeds[2].drain()
            health += _calls(health_calls, origin_wall)
            operates += _calls(operate_calls, origin_wall)
            ied_pids = {pid for pid, item in processes.items() if item["name"] == "studio_ied.exe"}
            backend = _totals(processes, lambda item: item["name"] in engine.backend_names)
            ied = _totals(processes, lambda item: item["name"] == "studio_ied.exe")
            other = _totals(processes, lambda item: item["name"].startswith("studio_") and item["name"] != "studio_ied.exe")
            cpu = {pid: item["cpu"] for pid, item in processes.items()}
            span = interval if previous is not None else None
            row = {"t": round(time.monotonic() - origin, 1), "free_mb": free_mb(),
                   "backend_ws_mb": backend[0], "backend_private_mb": backend[1], "backend_handles": backend[2],
                   "backend_threads": backend[3], "ied_ws_mb": ied[0], "ied_private_mb": ied[1], "ied_handles": ied[2],
                   "ied_threads": ied[3], "ied_workers": ied[4], "other_native_ws_mb": other[0],
                   "other_natives": sorted({item["name"] for item in processes.values()
                                            if item["name"].startswith("studio_") and item["name"] != "studio_ied.exe"}),
                   "backend_cpu_pct": _cpu(processes, cpu_before, span, lambda item: item["name"] in engine.backend_names),
                   "ied_cpu_pct": _cpu(processes, cpu_before, span, lambda item: item["name"] == "studio_ied.exe"),
                   "data_mb": _folder_mb(engine.data), "stdout_kb": round(sum(len(line) for line in engine.lines) / 1024, 1),
                   "stdout_lines": len(engine.lines), "sse_signals": kinds.get("signal", 0),
                   "sse_kinds": kinds, "sse_lag_p50_ms": _ms(percentile(lags, 0.5)), "sse_lag_p99_ms": _ms(percentile(lags, 0.99)),
                   "sse_disconnects": sse.disconnects, "sse_id_jumps": sse.id_jumps,
                   "health_n": len(health_calls), "health_errors": sum(1 for call in health_calls if call[2]),
                   "health_max_ms": _ms(max((call[1] for call in health_calls), default=None)),
                   "operate_n": len(operate_calls), "operate_errors": [call[2] for call in operate_calls if call[2]][:3],
                   "operate_max_ms": _ms(max((call[1] for call in operate_calls), default=None)),
                   "ied_pids_gone": sorted(previous - ied_pids) if previous is not None else [],
                   "ied_pids_new": sorted(ied_pids - previous) if previous is not None else [],
                   "runtime": bench_project.project(engine).get("runtime"), "processes": processes}
            previous, cpu_before = ied_pids, cpu
            rows.append(row)
            if out:
                out.write(json.dumps(row) + "\n")
                out.flush()
            if row["free_mb"] < min_free_mb:
                ended = f"stopped early: free RAM {row['free_mb']} MB < {min_free_mb} MB"
                break
        duration = rows[-1]["t"] if rows else 0
        light = [{key: value for key, value in row.items() if key != "processes"} for row in rows]
        return {"scenario": "SOAK", "ieds": ieds, "points_per_ied": points, "sources": configured,
                "seconds_planned": seconds, "seconds_run": duration, "ended": ended, "interval_s": interval,
                "operate_every_s": operate_every, "health_every_s": health_every,
                "trends": soak_report.trends(light), "operate_drift": soak_report.drift(operates, duration),
                "health_drift": soak_report.drift(health, duration),
                "operate_errors": [call for call in operates if call[2]][:20],
                "health_errors": [call for call in health if call[2]][:20],
                "ied_pids_gone_total": sum(len(row["ied_pids_gone"]) for row in rows),
                "ied_pids_new_total": sum(len(row["ied_pids_new"]) for row in rows),
                "sse_disconnects": rows[-1]["sse_disconnects"] if rows else None,
                "sse_id_jumps": rows[-1]["sse_id_jumps"] if rows else None,
                "runtime_states_seen": sorted({row["runtime"] for row in rows}), "rows": light,
                "push_topics": {"asked": list(push_topics), "answer": sse.topic_answer} if push_topics else None}
    finally:
        for feed in feeds:
            feed.close()
        if out:
            out.close()
        engine.stop()


def _cpu(processes, before, span, select):
    if not span:
        return None
    used = sum(item["cpu"] - before.get(pid, item["cpu"]) for pid, item in processes.items() if select(item))
    return round(100 * used / span, 2)


def _ms(value):
    return None if value is None else round(value * 1000, 2)
