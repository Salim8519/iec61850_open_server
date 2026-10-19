"""Native IEC 61850 speed evidence (workstream G): MMS multi-client latency, control round trip,
GOOSE retransmission gaps and idle CPU of simulated IED workers.

Usage: python studio/tools/native_speed_bench.py BIN_DIR OUT.json [--quick]
BIN_DIR holds studio_ied.exe and studio_test_mms_bench.exe of the build under test.
Addresses 127.67.x.y, ports >= 20000, temp folders and synthetic/public SCL only.
"""
import json
import socket
import statistics
import sys
import tempfile
import time

from native_bench_rig import GooseSink, Worker, bench, cpu_seconds, goose_model, start_bench, template_model

READ_REF, READ_FC = "BenchCTRL/XCBR1.Pos.stVal", "ST"
CLIENT_MIXES = [(1, 0), (1, 1), (1, 2), (1, 4), (2, 0), (3, 0), (5, 0)]


def read_latency(binary, folder, count):
    worker = Worker(binary, template_model("Bench"), folder, "127.67.0.10", 20010)
    try:
        bench(binary, worker.ip, worker.port, "read", READ_REF, READ_FC, 1, 0, 50)  # warm-up
        return [{"active": active, "idle": idle,
                 **bench(binary, worker.ip, worker.port, "read", READ_REF, READ_FC, active, idle, count)["all"]}
                for active, idle in CLIENT_MIXES]
    finally:
        worker.close()


def control_round_trip(binary, folder, count):
    results = []
    for enhanced in (False, True):
        name = "BenchSboE" if enhanced else "BenchSbo"
        worker = Worker(binary, template_model(name, enhanced), folder, "127.67.0.11", 20011)
        try:
            for idle in (0, 2):
                outcome = bench(binary, worker.ip, worker.port, "control", f"{name}CTRL/XCBR1.Pos", "CO",
                                1, idle, count)
                results.append({"model": "sbo_enhanced" if enhanced else "sbo_normal", "idle": idle,
                                "ok": outcome["ok"], **outcome["all"]})
        finally:
            worker.close()
    return results


def gaps_ms(times):
    return [round((later - earlier) * 1000, 2) for earlier, later in zip(times, times[1:])]


def goose_gaps(binary, folder, readers, idle=0):
    """Toggle XCBR1.Pos 5 times; the first frames after each change must follow MinTime 4 ms."""
    sink = GooseSink()
    worker = Worker(binary, goose_model(), folder, "127.67.0.12", 20012, sink.interface())
    load = None
    try:
        time.sleep(2.5)
        if readers:
            load = start_bench(binary, worker.ip, worker.port, "IED1_XCBRGenericIO/XCBR1.Pos.stVal", "ST",
                               readers, idle, 400000)
            time.sleep(1.0)
        bursts = []
        for index in range(5):
            mark = time.perf_counter()
            worker.request("update", ref="IED1_XCBRGenericIO/XCBR1.Pos.stVal", value=2 if index % 2 == 0 else 1)
            time.sleep(1.5)
            arrived = [stamp for stamp, _ in sink.frames if stamp >= mark]
            bursts.append({"first_ms": round((arrived[0] - mark) * 1000, 2) if arrived else None,
                           "gaps_ms": gaps_ms(arrived[:5])})
        steady = gaps_ms([stamp for stamp, _ in sink.frames])
        steady = [gap for gap in steady if gap > 100]
        burst_gaps = [gap for burst in bursts for gap in burst["gaps_ms"] if gap < 100]
        late = [gap for gap in burst_gaps if gap > 12]  # TAL of burst frames is 3 x MinTime
        return {"mms_readers": readers, "mms_idle": idle, "bursts": bursts, "burst_gap_max_ms": max(burst_gaps, default=None),
                "burst_gap_mean_ms": round(statistics.mean(burst_gaps), 2) if burst_gaps else None,
                "late_over_tal": len(late), "steady_gaps_ms": [min(steady, default=0), max(steady, default=0)],
                "frames": len(sink.frames)}
    finally:
        if load:
            load.kill()
            load.wait()
        worker.close()
        sink.close()


def idle_cpu(binary, folder, count, seconds, goose, held_connections=0):
    """CPU of `count` running workers; each may hold idle TCP connections (a client that stays)."""
    sink = GooseSink() if goose else None
    model = goose_model() if goose else template_model("Idle")
    workers, held = [], []
    try:
        for index in range(count):
            workers.append(Worker(binary, model, folder, f"127.67.1.{index + 1}", 20100 + index,
                                  sink.interface(f"010ccd01{index + 16:04x}") if sink else None))
            held += [socket.create_connection((f"127.67.1.{index + 1}", 20100 + index))
                     for _ in range(held_connections)]
        time.sleep(3.0)
        before = [cpu_seconds(worker.process.process) for worker in workers]
        started = time.perf_counter()
        time.sleep(seconds)
        elapsed = time.perf_counter() - started
        after = [cpu_seconds(worker.process.process) for worker in workers]
        used = [(late - early) / elapsed * 100 for early, late in zip(before, after)]
        return {"workers": count, "goose": goose, "held_connections": held_connections, "seconds": seconds,
                "total_pct_of_core": round(sum(used), 2), "per_worker_pct": round(statistics.mean(used), 3),
                "max_worker_pct": round(max(used), 3)}
    finally:
        for connection in held:
            connection.close()
        for worker in workers:
            worker.close()
        if sink:
            sink.close()


def main(arguments):
    binary, output = arguments[0], arguments[1]
    quick = "--quick" in arguments
    with tempfile.TemporaryDirectory(prefix="g_native_") as folder:
        results = {"bin": binary, "when": time.strftime("%Y-%m-%d %H:%M:%S")}
        results["read_latency"] = read_latency(binary, folder, 100 if quick else 300)
        results["control"] = control_round_trip(binary, folder, 20 if quick else 100)
        results["goose"] = [goose_gaps(binary, folder, *mix) for mix in ((0, 0), (2, 0), (1, 2))]
        results["idle_cpu"] = [idle_cpu(binary, folder, 20, 10 if quick else 30, goose, held)
                               for goose, held in ((False, 0), (True, 0), (False, 1))]
    with open(output, "w", encoding="utf-8") as handle:
        json.dump(results, handle, indent=1)
    print(json.dumps(results, indent=1))
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
