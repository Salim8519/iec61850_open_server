"""S4 operate, S5 external MMS clients, S7 real-IED sessions; S6/S8 need a dedicated rig.

Independent evidence comes from studio_probe.exe (native client), never from the engine.
"""
import json
import subprocess
import threading
import time

from engine_bench import project as bench_project
from engine_bench.procstats import Sampler
from engine_bench.sse_clients import percentile

CB_REF = "CTRL/XCBR1.Pos"
STATUS_REF = CB_REF + ".stVal"


def _probe(engine, ied, *args):
    completed = subprocess.run([str(engine.native_bin / "studio_probe.exe"), ied["ip"], str(ied["port"]), *args],
                               capture_output=True, text=True, timeout=30)
    if completed.returncode != 0:
        raise RuntimeError(completed.stdout + completed.stderr)
    return json.loads(completed.stdout)


def _running(make_engine, ieds, points):
    engine = make_engine()
    bench_project.import_ieds(engine, ieds, points)
    bench_project.relocate(engine)
    engine.json("POST", "/api/v1/runtime/start")
    bench_project.wait_state(engine, "running")
    return engine, bench_project.project(engine)["ieds"]


def s4_operate(make_engine, operates=200, points=2000):
    """API operates on the sample breaker, each confirmed by an independent probe read."""
    engine, ieds = _running(make_engine, 1, points)
    try:
        ied, name = ieds[0], ieds[0]["name"]
        ref, latencies, confirmed = f"{name}{CB_REF}", [], 0
        for number in range(operates):
            value = number % 2 == 0
            started = time.perf_counter()
            engine.json("POST", "/api/v1/controls/operate", {"ied_id": ied["id"], "ref": ref, "value": value})
            latencies.append(time.perf_counter() - started)
            confirmed += _probe(engine, ied, "read", f"{name}{STATUS_REF}", "ST")["value"] == (2 if value else 1)
        return {"scenario": "S4", "operates": operates, "confirmed_by_probe": confirmed,
                "api_p50_ms": round(percentile(latencies, 0.5) * 1000, 2),
                "api_p95_ms": round(percentile(latencies, 0.95) * 1000, 2)}
    finally:
        engine.stop()


def s5_external_mms(make_engine, clients=1, reads=200, points=2000):
    """`clients` independent probes reading in parallel; read latency per request."""
    engine, ieds = _running(make_engine, 1, points)
    try:
        ied, latencies, errors = ieds[0], [], []
        ref = f"{ied['name']}{STATUS_REF}"

        def reader():
            for _ in range(reads // clients):
                started = time.perf_counter()
                try:
                    _probe(engine, ied, "read", ref, "ST")
                    latencies.append(time.perf_counter() - started)
                except (RuntimeError, subprocess.TimeoutExpired) as error:
                    errors.append(str(error)[:200])
        sampler = Sampler(engine.job, engine.backend_names).start()
        threads = [threading.Thread(target=reader) for _ in range(clients)]
        for thread in threads:
            thread.start()
        for thread in threads:
            thread.join()
        return {"scenario": "S5", "clients": clients, "reads": len(latencies), "errors": len(errors),
                "probe_p50_ms": round(percentile(latencies, 0.5) * 1000, 2) if latencies else None,
                "probe_p99_ms": round(percentile(latencies, 0.99) * 1000, 2) if latencies else None,
                "note": "includes studio_probe process start per read", "resources": sampler.stop()}
    finally:
        engine.stop()


def s7_real_ieds(make_engine, sessions=1, points=2000, seconds=30):
    """A second engine simulates the targets on 127.66.1.x; the first reads them as real IEDs."""
    targets = make_engine()
    reader = None
    try:
        bench_project.import_ieds(targets, sessions, points, prefix="Target")
        bench_project.relocate(targets, block=1)
        targets.json("POST", "/api/v1/runtime/start")
        bench_project.wait_state(targets, "running")
        reader = make_engine()
        bench_project.import_ieds(reader, sessions, points, prefix="Target")
        status, archive, _ = reader.request("GET", "/api/v1/project/download")
        _point_at(reader, archive, sessions)
        started = time.perf_counter()
        reader.json("POST", "/api/v1/runtime/start")
        bench_project.wait_state(reader, "running")
        connect = time.perf_counter() - started
        sampler = Sampler(reader.job, reader.backend_names).start()
        time.sleep(seconds)
        return {"scenario": "S7", "sessions": sessions, "start_seconds": round(connect, 3),
                "resources": sampler.stop()}
    finally:
        if reader:
            reader.stop()
        targets.stop()


def _point_at(reader, archive, sessions):
    import io
    import zipfile
    with zipfile.ZipFile(io.BytesIO(archive)) as source:
        document = json.loads(source.read("project.json"))
    for index, ied in enumerate(document["ieds"]):
        ied.update(role="real", ip=bench_project.address(index, 2), port=bench_project.BASE_PORT,
                   live={**(ied.get("live") or {}), "ip": bench_project.address(index, 1), "port": bench_project.BASE_PORT})
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as target:
        target.writestr("project.json", json.dumps(document))
    reader.upload("/api/v1/project/open", "file", "real.scadabench", stream.getvalue())


def needs_rig(scenario, reason):
    return {"scenario": scenario, "skipped": reason}


def s6_goose(*_args, **_kwargs):
    return needs_rig("S6", "GOOSE publishers need the Npcap rig (STUDIO_GOOSE_ADAPTER); extend "
                           "tests/protocol/test_goose_engine_latency.py on that rig")


def s8_gateway(*_args, **_kwargs):
    return needs_rig("S8", "the IEC 104 gateway listens beyond loopback and may need firewall rules; run only on "
                           "the lab rig with an external master, never on the owner's laptop")
