"""HAMMER: canvas breaker open/close as fast as the UI path allows, with N idle MMS clients.

Every command goes through POST /equipment/<node>/operate (the canvas endpoint). Every
`probe_every`th result is confirmed independently by studio_probe reading XCBR1.Pos.stVal
(Dbpos printed as bits: "10" closed, "01" open). Idle clients (mms_idle) hold real
associations on the same IED for the whole hammer, which is where ADR-0079 should differ.
"""
import collections
import json
import subprocess
import time

from engine_bench import canvas, project as bench_project
from engine_bench.mms_idle import IdleClients, capture_association
from engine_bench.pertype import by_type
from engine_bench.procstats import Sampler
from engine_bench.sse_clients import percentile

EXPECTED = {"close": "10", "open": "01"}


def probe_read(engine, ied, reference):
    started = time.perf_counter()
    completed = subprocess.run([str(engine.native_bin / "studio_probe.exe"), ied["ip"], str(ied["port"]),
                                "read", reference, "ST"], capture_output=True, text=True, timeout=30)
    elapsed = time.perf_counter() - started
    try:
        answer = json.loads(completed.stdout)
    except ValueError:
        answer = {"ok": False, "raw": (completed.stdout + completed.stderr)[:200]}
    return answer, elapsed


def latency_summary(values):
    if not values:
        return {}
    return {"count": len(values), "mean_ms": round(1000 * sum(values) / len(values), 2),
            "p50_ms": round(1000 * percentile(values, 0.5), 2), "p95_ms": round(1000 * percentile(values, 0.95), 2),
            "p99_ms": round(1000 * percentile(values, 0.99), 2), "max_ms": round(1000 * max(values), 2),
            "min_ms": round(1000 * min(values), 2)}


def hammer(make_engine, operates=500, points=2000, idle_clients=0, probe_every=10):
    engine = make_engine()
    idlers = None
    try:
        bench_project.import_ieds(engine, 1, points)
        bench_project.relocate(engine)
        ied = bench_project.project(engine)["ieds"][0]
        node = canvas.add_breakers(engine, [ied])[0]
        engine.json("POST", "/api/v1/runtime/start")
        bench_project.wait_state(engine, "running")
        status_ref = ied["name"] + canvas.STATUS
        if idle_clients:
            frames = capture_association(engine.native_bin / "studio_probe.exe", ied["ip"], ied["port"], status_ref)
            idlers = IdleClients(frames, ied["ip"], ied["port"], idle_clients)
            time.sleep(1)
        sampler = Sampler(engine.job, engine.backend_names).start()
        latencies, errors, probes, probe_latency = [], [], collections.Counter(), []
        confirmed = sampled = 0
        started = time.perf_counter()
        for number in range(operates):
            action = "close" if number % 2 == 0 else "open"
            elapsed, error = canvas.operate(engine, node, action)
            latencies.append(elapsed)
            if error:
                errors.append({"n": number, "action": action, "error": error})
            if number % probe_every == (number // probe_every) % 2:  # alternate close/open samples
                answer, took = probe_read(engine, ied, status_ref)
                probe_latency.append(took)
                sampled += 1
                probes[f"{action}->{answer.get('value')}"] += 1
                confirmed += answer.get("value") == EXPECTED[action]
        wall = time.perf_counter() - started
        rows = list(sampler.rows)
        resources = sampler.stop()
        idle_state = None
        if idlers:
            idle_state = {"requested": idle_clients, "accepted": idlers.accepted,
                          "closed_by_server_during_run": idlers.closed_by_server}
        return {"scenario": "HAMMER", "operates": operates, "points_per_ied": points, "idle_clients": idle_state,
                "endpoint": "POST /api/v1/equipment/<node>/operate", "wall_seconds": round(wall, 2),
                "operates_per_second": round(operates / wall, 2), "errors": len(errors), "error_samples": errors[:10],
                "latency": latency_summary(latencies), "probe_sampled": sampled, "probe_confirmed": confirmed,
                "probe_values": dict(probes), "probe_read_latency": latency_summary(probe_latency),
                "resources": resources, "per_type": by_type(rows)}
    finally:
        if idlers:
            idlers.close()
        engine.stop()
