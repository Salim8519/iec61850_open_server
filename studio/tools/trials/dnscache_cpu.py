"""Windows DNS Client (Dnscache svchost) CPU for 30 s after Lab-card address adds and removes.

    python studio/tools/trials/dnscache_cpu.py --repeats 3

Change 0208 saw the DNS Client keep 5-11 processors busy for 20-30 s after each address change
on a real card. Here a separate instance (small project on the Lab card) measures, after each
action: one IED started alone (one address added), that IED stopped (one address removed),
a full Run (every address added) and a Stop (every address removed), plus an idle baseline.
CPU comes from the system process list, so no administrator rights are needed.
"""
import argparse
import json
from pathlib import Path
import sys
import time

sys.path.insert(0, str(Path(__file__).resolve().parent))
import projects  # noqa: E402
from instance import Instance  # noqa: E402
from winproc import cpu_seconds, service_pid, system_busy  # noqa: E402

RESULTS = projects.REPO / ".local/trials/results"


def sample(pid, seconds, step=0.5):
    """CPU seconds used in `seconds`, the busiest `step` window in cores, machine busy share."""
    start, busy = cpu_seconds([pid])[pid], system_busy()
    previous, peak, deadline = start, 0.0, time.perf_counter() + seconds
    while time.perf_counter() < deadline:
        time.sleep(step)
        current = cpu_seconds([pid])[pid]
        peak, previous = max(peak, (current - previous) / step), current
    end = system_busy()
    return {"cpu_seconds": round(previous - start, 3), "average_cores": round((previous - start) / seconds, 3),
            "peak_cores": round(peak, 2), "machine_busy_percent": round(100 * (end[0] - busy[0]) / (end[1] - busy[1]), 1)}


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--repeats", type=int, default=3)
    parser.add_argument("--seconds", type=float, default=30.0)
    parser.add_argument("--shared", action="store_true", help="start although another engine runs (see README)")
    args = parser.parse_args()
    pid = service_pid("Dnscache")
    results = {"idle": [], "one_add": [], "one_remove": [], "run_add": [], "stop_remove": []}
    with Instance(shared=args.shared) as engine:
        projects.open_in(engine, projects.build("small"))
        projects.use_lab_card(engine)
        first = engine.json("GET", "/api/v1/project")["ieds"][0]["id"]
        time.sleep(8)
        actions = [("idle", None), ("one_add", f"/api/v1/runtime/ieds/{first}/start"),
                   ("one_remove", f"/api/v1/runtime/ieds/{first}/stop"),
                   ("run_add", "/api/v1/runtime/start"), ("stop_remove", "/api/v1/runtime/stop")]
        for repeat in range(args.repeats):
            for name, path in actions:
                try:
                    if path:
                        engine.json("POST", path)
                except RuntimeError as error:  # e.g. another engine took our address off the card
                    results.setdefault("refused", []).append({"action": name, "error": str(error)})
                    print(f"{repeat + 1} {name}: refused: {error}", flush=True)
                    continue
                results[name].append(sample(pid, args.seconds))
                print(f"{repeat + 1} {name}: {results[name][-1]}", flush=True)
    RESULTS.mkdir(parents=True, exist_ok=True)
    out = RESULTS / f"dnscache-{time.strftime('%Y%m%d-%H%M%S')}.json"
    out.write_text(json.dumps({"seconds": args.seconds, "results": results}, indent=1), encoding="utf-8")
    print(f"saved {out}")


if __name__ == "__main__":
    main()
