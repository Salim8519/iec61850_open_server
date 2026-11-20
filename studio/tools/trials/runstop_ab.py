"""Run and Stop trials R1, R2, R3, R5: today's code against a trial build, alternating.

    set SCADABENCH_TRIAL_RANGE=198.18.26
    set SCADABENCH_TRIAL_PORTS=26000-26999
    python studio/tools/trials/runstop_ab.py --project large --rounds 10 --per-session 2 ^
        --base studio/build/engine/studio_server_base.exe=<native Release dir> ^
        --trial studio/build/engine/studio_server_r1.exe=<native trial dir> --label r1

Each side is "server exe=native folder". Sessions alternate (base, trial, base, ...): a
session starts a separate engine instance (temp data folder), opens the project, puts every
IED on the Lab card, waits --warm seconds, then does --per-session Run/Stop rounds with
step0_run_stop's measurements. Rounds in which Windows showed our addresses vanish (another
engine's Release) or whose Run was refused are listed and left out of the medians.
Writes .local/trials/results/ab-<label>-<project>-<time>.json and prints both summaries.
"""
import argparse
import json
from pathlib import Path
import sys
import time

sys.path.insert(0, str(Path(__file__).resolve().parent))
import projects  # noqa: E402
import summary  # noqa: E402
import timeline  # noqa: E402
from instance import Instance  # noqa: E402
from sse import EventRecorder  # noqa: E402
from step0_run_stop import RESULTS, one_run, one_stop  # noqa: E402
from watcher import Watcher  # noqa: E402
from winproc import system_busy  # noqa: E402


def side(text):
    server, native = text.split("=", 1)
    return {"server": str(Path(server).resolve()), "native": str(Path(native).resolve())}


def session(config, document, rounds, hold, gap, warm, into):
    with Instance(server=config["server"], native=config["native"], shared=True) as engine:
        projects.open_in(engine, document)
        projects.use_lab_card(engine)
        events, watch = EventRecorder(engine).start(), Watcher(engine.job).start()
        time.sleep(warm)
        try:
            for _ in range(rounds):
                run = one_run(engine, watch, events, settle=hold)
                if "error" in run:
                    into["failed_runs"].append(run)
                    print(f"  Run refused: {run['error'][:200]}", flush=True)
                    time.sleep(gap)
                    continue
                # R2: the order of the initial-state writes (IED, reference) as the events show it
                run["facts"]["initial_order"] = [[e.get("ied_id"), e.get("ref")] for e in events.since(run["facts"]["click"])
                                                 if e.get("kind") == "signal" and e.get("source") == "initial_state"]
                stop = one_stop(engine, watch, events, settle=1.0)
                stop["facts"]["stopped_shown"] = stop["facts"].get("stopped_event")
                # Another engine took our addresses off while running (since ADR-0141 the
                # addresses stay after Stop, so none leaving at Stop is normal).
                if run["facts"]["disturbed"]:
                    into["disturbed_rounds"].append({"run": run, "stop": stop})
                else:
                    into["runs"].append(run)
                    into["stops"].append(stop)
                time.sleep(gap)
        finally:
            watch.stop()
            events.stop()


def describe(name, data):
    runs, stops = data["runs"], data["stops"]
    return {"side": name, "rounds": len(runs), "failed_runs": len(data["failed_runs"]),
            "disturbed": len(data["disturbed_rounds"]),
            "run": summary.describe(runs, timeline.RUN_STEPS), "stop": summary.describe(stops, timeline.STOP_STEPS),
            "running_event": summary.fact(runs, "running_event"), "stopped_event": summary.fact(stops, "stopped_event")}


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--project", choices=sorted(projects.SOURCES), default="small")
    parser.add_argument("--base", type=side, required=True, help="server.exe=native folder (today's code)")
    parser.add_argument("--trial", type=side, required=True, help="server.exe=native folder (the trial)")
    parser.add_argument("--label", default="trial")
    parser.add_argument("--rounds", type=int, default=10, help="counted rounds wanted per side")
    parser.add_argument("--per-session", type=int, default=2)
    parser.add_argument("--max-sessions", type=int, default=20, help="per side, disturbed rounds included")
    parser.add_argument("--hold", type=float, default=2.0)
    parser.add_argument("--gap", type=float, default=3.0)
    parser.add_argument("--warm", type=float, default=8.0)
    args = parser.parse_args()
    document = projects.build(args.project)
    data = {name: {"runs": [], "stops": [], "failed_runs": [], "disturbed_rounds": [], "sessions": 0}
            for name in ("base", "trial")}
    busy_start = system_busy()
    while any(len(d["runs"]) < args.rounds and d["sessions"] < args.max_sessions for d in data.values()):
        for name in ("base", "trial"):
            entry = data[name]
            if len(entry["runs"]) >= args.rounds or entry["sessions"] >= args.max_sessions:
                continue
            entry["sessions"] += 1
            try:
                session(getattr(args, name), document, args.per_session, args.hold, args.gap, args.warm, entry)
            except RuntimeError as error:  # an instance that did not start or open: next session
                entry.setdefault("session_errors", []).append(str(error)[:300])
                print(f"  session failed: {str(error)[:200]}", flush=True)
            last = entry["runs"][-1]["facts"]["total"] if entry["runs"] else None
            print(f"{name} session {entry['sessions']}: {len(entry['runs'])} counted, last run {last}, "
                  f"disturbed {len(entry['disturbed_rounds'])}, refused {len(entry['failed_runs'])}", flush=True)
    busy_end = system_busy()
    result = {"label": args.label, "project": args.project, "ieds": len(document["ieds"]),
              "range": projects.RANGE, "base": vars(args)["base"], "trial": vars(args)["trial"],
              "machine_busy_percent": round(100 * (busy_end[0] - busy_start[0]) / (busy_end[1] - busy_start[1]), 1),
              "summary": [describe(name, data[name]) for name in ("base", "trial")], "raw": data}
    RESULTS.mkdir(parents=True, exist_ok=True)
    path = RESULTS / f"ab-{args.label}-{args.project}-{time.strftime('%Y%m%d-%H%M%S')}.json"
    path.write_text(json.dumps(result, indent=1), encoding="utf-8")
    for entry in result["summary"]:
        print(f"\n{entry['side']}: {entry['rounds']} rounds, disturbed {entry['disturbed']}, refused "
              f"{entry['failed_runs']}\nRun\n{summary.markdown(entry['run'])}\nStop\n{summary.markdown(entry['stop'])}"
              f"\nrunning event {entry['running_event']}, stopped event {entry['stopped_event']}")
    print(f"\nmachine busy {result['machine_busy_percent']} %, saved {path}")


if __name__ == "__main__":
    main()
