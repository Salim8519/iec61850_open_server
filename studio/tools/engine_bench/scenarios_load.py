"""S2 generated updates and S10 browser fan-out: delivered rate, lag, lateness, resources."""
import time

from engine_bench import project as bench_project
from engine_bench.procstats import Sampler
from engine_bench.sse_clients import SseClient, percentile

RATE_HZ = 10


def _float_refs(engine, ied_id, count):
    page = engine.json("GET", f"/api/v1/ieds/{ied_id}/signals?limit=5000")
    return [item["ref"] for item in page["items"] if item["kind"] == "FLOAT32"][:count]


def _configure_sources(engine, sources):
    ieds = bench_project.project(engine)["ieds"]
    per_ied, configured = max(1, sources // len(ieds)), 0
    for ied in ieds:
        for ref in _float_refs(engine, ied["id"], per_ied):
            if configured >= sources:
                return configured
            engine.json("POST", "/api/v1/signals/source", {"ied_id": ied["id"], "ref": ref, "spec": {
                "mode": "ramp", "offset": 100, "amplitude": 20, "period": 5, "seed": 1, "overridden": False}})
            configured += 1
    return configured


def s2_generators(make_engine, ieds=2, points=2000, sources=100, clients=1, seconds=60):
    engine = make_engine()
    readers = []
    try:
        bench_project.import_ieds(engine, ieds, points)
        bench_project.relocate(engine)
        configured = _configure_sources(engine, sources)
        engine.json("POST", "/api/v1/runtime/start")
        bench_project.wait_state(engine, "running")
        readers = [SseClient(engine).start() for _ in range(clients)]
        time.sleep(2)  # settle: workers up, first ticks out
        marks = [reader.mark() for reader in readers]
        sampler = Sampler(engine.job, engine.backend_names).start()
        started = time.perf_counter()
        time.sleep(seconds)
        elapsed = time.perf_counter() - started
        resources = sampler.stop()
        target = configured * RATE_HZ
        delivered = [(reader.signals - mark) / elapsed for reader, mark in zip(readers, marks)]
        lags = [lag for reader in readers for lag in reader.lags]
        return {"scenario": "S2" if clients <= 1 else "S10", "ieds": ieds, "points_per_ied": points,
                "sources": configured, "clients": clients, "seconds": seconds, "target_per_s": target,
                "delivered_per_s": [round(rate, 1) for rate in delivered],
                "delivered_ratio_min": round(min(delivered) / target, 4) if target and delivered else None,
                "lag_p50_ms": _ms(percentile(lags, 0.5)), "lag_p99_ms": _ms(percentile(lags, 0.99)),
                "resyncs": sum(reader.resyncs for reader in readers), "skipped_ticks": max((reader.skipped_ticks for reader in readers), default=0),
                "resources": resources}
    finally:
        for reader in readers:
            reader.close()
        engine.stop()


def s10_fanout(make_engine, ieds=2, points=2000, sources=100, tabs=(1, 3), seconds=30):
    return [s2_generators(make_engine, ieds, points, sources, clients, seconds) for clients in tabs]


def _ms(value):
    return None if value is None else round(value * 1000, 2)
