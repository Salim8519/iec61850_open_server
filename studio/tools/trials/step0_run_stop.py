"""Step 0: time every step of Run and Stop on today's code (speed-trials plan 2026-11-20).

    python studio/tools/trials/step0_run_stop.py --project small --rounds 10
    python studio/tools/trials/step0_run_stop.py --project large --rounds 10 --goose off

Starts a separate engine instance (temp data folder, port >= 20000), opens the project, puts
every IED on the ScadaBench Lab card (198.18.20.x), then Run/Stop `rounds` times. Each round is
split into steps by timeline.py from Windows' own records and the engine's events. Writes the
raw rounds and the summary to .local/trials/results/ and prints Markdown tables.
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
from instance import Instance, busy_reasons  # noqa: E402
from sse import EventRecorder  # noqa: E402
from watcher import Watcher  # noqa: E402
from winclock import now  # noqa: E402
from winproc import system_busy  # noqa: E402

RESULTS = projects.REPO / ".local/trials/results"


def failed_run(engine, watch, events, click, error):
    """A refused Run is a finding, not a harness crash: keep Windows' view of the moment."""
    time.sleep(1.0)
    return {"error": str(error), "seconds": round(now() - click, 3), "click": click,
            "removed_by_us": {ip: round(t - click, 4) for ip, t in timeline.journal(engine.data, "removed", click).items()},
            "windows": {ip: [row.state, round(row.created - click, 4)] for ip, row in watch.addresses.items()},
            "listening": {f"{ip}:{port}": pid for (ip, port), pid in watch.listening.items()},
            "address_changes": [[round(t - click, 4), ip, change] for t, ip, change in watch.changes("address")],
            "applied": {ip: round(t - click, 4) for ip, t in timeline.journal(engine.data, "applied", click).items()},
            "events": [{k: e.get(k) for k in ("kind", "state", "message", "timestamp")}
                       for e in events.since(click) if e.get("kind") in ("runtime", "listen_check_failed")]}


def one_run(engine, watch, events, settle):
    watch.mark()
    click = now()
    try:
        engine.json("POST", "/api/v1/runtime/start")
    except RuntimeError as error:
        return failed_run(engine, watch, events, click, error)
    answered = now()
    time.sleep(settle)  # late events and the watcher's last poll
    steps, facts = timeline.run_steps(click, answered, events.since(click),
                                      timeline.journal(engine.data, "applied", click),
                                      watch.kernel_times(since=click), watch.first("listen", "added"),
                                      projects.IED_PORT)
    facts["windows_addresses"] = len(watch.windows_created())
    facts["click"] = click
    # Another engine's Release (its launch, Stop or Use) takes our addresses off the shared card.
    facts["disturbed"] = bool(watch.first("address", "removed"))
    return {"steps": steps, "facts": facts}


def one_stop(engine, watch, events, settle):
    watch.mark()
    click = now()
    engine.json("POST", "/api/v1/runtime/stop")
    answered = now()
    time.sleep(settle)
    steps, facts = timeline.stop_steps(click, answered, events.since(click),
                                       timeline.journal(engine.data, "removed", click),
                                       watch.kernel_times(since=click), watch.first("address", "removed"))
    facts["windows_addresses_left"] = len(watch.windows_created())
    facts["click"] = click
    return {"steps": steps, "facts": facts}


def measure(name, rounds, goose, hold, gap, warm, shared=False):
    document = projects.build(name)
    if goose == "off":
        for ied in document["ieds"]:
            ied["goose_publish"] = False
    others = busy_reasons()  # recorded with the result: a shared run is not a quiet-machine run
    with Instance(shared=shared) as engine:
        projects.open_in(engine, document)
        projects.use_lab_card(engine)
        events, watch = EventRecorder(engine).start(), Watcher(engine.job).start()
        time.sleep(warm)  # opening's idle work (start data, spare GOOSE engine) settles first
        busy_start = system_busy()
        runs, stops, failures, disturbed = [], [], [], []
        try:
            for index in range(rounds):
                run = one_run(engine, watch, events, settle=hold)
                if "error" in run:
                    failures.append(run)
                    print(f"round {index + 1}: Run refused: {run['error']}", flush=True)
                    time.sleep(gap)
                    continue
                stop = one_stop(engine, watch, events, settle=1.0)
                if run["facts"]["disturbed"]:
                    disturbed.append({"run": run, "stop": stop})
                    print(f"round {index + 1}: our addresses were removed while running; round not counted", flush=True)
                    time.sleep(gap)
                    continue
                runs.append(run)
                stops.append(stop)
                print(f"round {index + 1}: run {runs[-1]['facts']['total']:.3f} s, "
                      f"stop {stops[-1]['facts']['total']:.3f} s", flush=True)
                time.sleep(gap)
        finally:
            watch.stop()
            events.stop()
        busy_end = system_busy()
    load = (busy_end[0] - busy_start[0]) / (busy_end[1] - busy_start[1])
    return {"project": name, "ieds": len(document["ieds"]), "goose": goose, "rounds": rounds,
            "machine_busy_percent": round(100 * load, 1), "other_engines": others,
            "runs": runs, "stops": stops, "failed_runs": failures, "disturbed_rounds": disturbed,
            "run_summary": summary.describe(runs, timeline.RUN_STEPS),
            "stop_summary": summary.describe(stops, timeline.STOP_STEPS),
            "running_event": summary.fact(runs, "running_event"),
            "stopped_event": summary.fact(stops, "stopped_event")}


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--project", choices=sorted(projects.SOURCES), default="small")
    parser.add_argument("--rounds", type=int, default=10)
    parser.add_argument("--goose", choices=["on", "off"], default="on", help="off: every IED's GOOSE publishing off")
    parser.add_argument("--hold", type=float, default=2.0, help="seconds running before Stop")
    parser.add_argument("--gap", type=float, default=3.0, help="seconds stopped before the next Run")
    parser.add_argument("--warm", type=float, default=8.0, help="seconds after opening before round 1")
    parser.add_argument("--shared", action="store_true", help="start although another engine runs (see README)")
    args = parser.parse_args()
    result = measure(args.project, args.rounds, args.goose, args.hold, args.gap, args.warm, args.shared)
    RESULTS.mkdir(parents=True, exist_ok=True)
    path = RESULTS / f"step0-{args.project}-goose{args.goose}-{time.strftime('%Y%m%d-%H%M%S')}.json"
    path.write_text(json.dumps(result, indent=1), encoding="utf-8")
    print(f"\n{result['project']}: {result['ieds']} IEDs, GOOSE {args.goose}, {args.rounds} rounds, "
          f"machine busy {result['machine_busy_percent']} %, refused Runs {len(result['failed_runs'])}, disturbed {len(result['disturbed_rounds'])}\n\nRun\n{summary.markdown(result['run_summary'])}"
          f"\n\nStop\n{summary.markdown(result['stop_summary'])}\n\nrunning event {result['running_event']}, "
          f"stopped event {result['stopped_event']}\nsaved {path}")


if __name__ == "__main__":
    main()
