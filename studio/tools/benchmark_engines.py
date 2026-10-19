"""Benchmark either engine over HTTP only: scenarios S1-S10 of the Go engine capacity audit,
plus stress runs SCALE, HAMMER, CYCLES, APPCYCLES and SOAK (owner stress test 2026-10-20).

    python studio/tools/benchmark_engines.py --engine python --scenarios S1 --ieds 1 --points 5000 \
        --seconds 60 --native-bin <folder with studio_*.exe> --out evidence.json

Each run starts the product launcher (no dev watcher), puts IEDs on 127.66.x.y with high
ports, samples backend and child processes every second (working set, private bytes, CPU,
handles) and stores provenance (source hashes, Python/dependency versions, machine).
Scale scenarios accept the same knobs; on a small laptop keep --ieds 1-2 (owner rule).
"""
import argparse
import json
import os
from pathlib import Path
import platform
import sys
import time

STUDIO = Path(__file__).resolve().parents[1]
DEFAULT_BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", STUDIO / "native/build/Release"))


def _paths():
    for path in (STUDIO / "tools", STUDIO / "tests", STUDIO / "backend"):
        if str(path) not in sys.path:
            sys.path.insert(0, str(path))


def scenarios():
    from engine_bench import scenarios_load as load, scenarios_run as run, scenarios_wire as wire
    from engine_bench import scenarios_hammer as hammer, scenarios_lifecycle as life, scenarios_scale as scale
    from engine_bench import scenarios_soak as soak
    from engine_bench.engine import Engine
    return {"S1": lambda make, a: run.s1_idle(make, a.ieds, a.points, a.seconds),
            "S2": lambda make, a: load.s2_generators(make, a.ieds, a.points, a.sources, 1, a.seconds),
            "S3": lambda make, a: run.s3_scale(make, a.ieds, a.points, a.seconds),
            "S4": lambda make, a: wire.s4_operate(make, a.operates, a.points),
            "S5": lambda make, a: wire.s5_external_mms(make, a.clients, a.operates, a.points),
            "S6": wire.s6_goose,
            "S7": lambda make, a: wire.s7_real_ieds(make, a.ieds, a.points, a.seconds),
            "S8": wire.s8_gateway,
            "S9": lambda make, a: run.s9_project_ops(make, a.ieds, a.points, a.repeats),
            "S10": lambda make, a: load.s10_fanout(make, a.ieds, a.points, a.sources,
                                                   tuple(int(n) for n in a.tabs.split(",")), a.seconds),
            "HAMMER": lambda make, a: hammer.hammer(make, a.operates, a.points, a.idle_clients, a.probe_every),
            "SCALE": lambda make, a: scale.scale(make, tuple(int(n) for n in a.levels.split(",")), a.points, a.seconds),
            "CYCLES": lambda make, a: life.run_cycles(make, a.ieds, a.points, a.cycles),
            "APPCYCLES": lambda make, a: life.app_cycles(make, lambda: Engine(a.engine, a.native_bin), a.ieds,
                                                         a.points, a.cycles),
            "SOAK": lambda make, a: soak.soak(make, a.ieds, a.points, a.sources, a.seconds, a.interval,
                                              a.operate_every, stream=a.stream)}


def provenance(native_bin):
    from benchmark_provenance import capture
    import hashlib
    identity = capture(STUDIO)
    natives = {path.name: hashlib.sha256(path.read_bytes()).hexdigest()
               for path in sorted(Path(native_bin).glob("studio_*.exe"))}
    return {"source_sha256": identity["sha256"], "python": identity["python"], "platform": identity["platform"],
            "dependencies": identity["dependencies"], "natives": natives, "machine": platform.processor(),
            "logical_cpus": os.cpu_count(), "recorded_at": time.strftime("%Y-%m-%dT%H:%M:%S%z")}


def parse(argv=None):
    parser = argparse.ArgumentParser(description="ScadaBench engine benchmarks S1-S10 (HTTP only)")
    parser.add_argument("--engine", choices=["python", "go"], default="python")
    parser.add_argument("--scenarios", default="S1")
    parser.add_argument("--native-bin", type=Path, default=DEFAULT_BIN)
    parser.add_argument("--ieds", type=int, default=1)
    parser.add_argument("--points", type=int, default=5000)
    parser.add_argument("--seconds", type=int, default=60)
    parser.add_argument("--sources", type=int, default=100)
    parser.add_argument("--operates", type=int, default=200)
    parser.add_argument("--clients", type=int, default=1)
    parser.add_argument("--repeats", type=int, default=5)
    parser.add_argument("--tabs", default="1,3,10")
    parser.add_argument("--idle-clients", type=int, default=0, help="HAMMER: idle MMS associations held")
    parser.add_argument("--probe-every", type=int, default=10, help="HAMMER: confirm every Nth command by probe")
    parser.add_argument("--levels", default="1,5,10,20", help="SCALE: IED counts")
    parser.add_argument("--cycles", type=int, default=50, help="CYCLES/APPCYCLES: repetitions")
    parser.add_argument("--interval", type=int, default=10, help="SOAK: sample interval (s)")
    parser.add_argument("--operate-every", type=float, default=5, help="SOAK: seconds between canvas commands")
    parser.add_argument("--stream", type=Path, help="SOAK: append one JSON line per sample here")
    parser.add_argument("--base-port", type=int, default=18700, help="IED port on 127.66.x.y (owner rule: >= 20000)")
    parser.add_argument("--out", type=Path)
    return parser.parse_args(argv)


def main(argv=None):
    args = parse(argv)
    _paths()
    from engine_bench.engine import Engine
    from engine_bench import project as bench_project
    bench_project.BASE_PORT = args.base_port
    if not (args.native_bin / "studio_ied.exe").is_file():
        raise SystemExit(f"native workers not found in {args.native_bin} (--native-bin or STUDIO_NATIVE_BIN)")
    table = scenarios()
    evidence = {"engine": args.engine, "arguments": {key: str(value) for key, value in vars(args).items()
                              if key not in ("native_bin", "out")},  # no local paths in evidence
                "provenance": provenance(args.native_bin), "results": []}
    for name in args.scenarios.split(","):
        result = table[name.strip().upper()](lambda: Engine(args.engine, args.native_bin).start(), args)
        evidence["results"].append(result)
        print(json.dumps(result, indent=2), flush=True)
    if args.out:
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    return 0


if __name__ == "__main__":
    sys.exit(main())
