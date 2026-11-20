"""Trial N10 (speed and network trials 2026-11-20, change 0222): IED addresses kept on the card for
the session instead of added at every Run and removed at every Stop.

    python studio/tools/trials/n10_session_addresses.py --rounds 10 --server base
    python studio/tools/trials/n10_session_addresses.py --rounds 10 --server trial --project large

Part 1, timing: the project on the Lab card (198.18.24.x), `rounds` Run/Stop rounds cut into
steps exactly as step0_run_stop.py does, with the CPU the Windows DNS Client used over the rounds.
Part 2, nothing left behind: Run + Stop then close the project (File › Close), Run + Stop then
exit, and Run + Stop then End task (kill) and launch again. Windows' list of the Lab card is read
after each. Result: .local/trials/results/n10-*.json.
"""
import argparse
from pathlib import Path
import sys
import time

sys.path.insert(0, str(Path(__file__).resolve().parent))
import labgroup as g  # noqa: E402
import projects  # noqa: E402
import step0_run_stop as step0  # noqa: E402
import summary  # noqa: E402
import timeline  # noqa: E402
from sse import EventRecorder  # noqa: E402
from watcher import Watcher  # noqa: E402
from winproc import cpu_seconds, service_pid, system_busy  # noqa: E402


def timing(server, name, rounds, hold=2.0, gap=3.0, warm=8.0):
    dns = service_pid("Dnscache")
    engine = g.LabInstance(server=server).start()
    try:
        projects.open_in(engine, projects.build(name))
        projects.use_lab_card(engine)
        events, watch = EventRecorder(engine).start(), Watcher(engine.job, prefix=g.PREFIX).start()
        time.sleep(warm)
        busy, dns_start = system_busy(), cpu_seconds([dns])[dns]
        runs, stops, refused, disturbed, adds, removes = [], [], [], 0, 0, 0
        for index in range(rounds):
            run = step0.one_run(engine, watch, events, settle=hold)
            if "error" in run:
                refused.append(run["error"][:300])
                time.sleep(gap)
                continue
            adds += len(watch.first("address", "added"))  # Windows' own list, since this Run's click
            stop = step0.one_stop(engine, watch, events, settle=1.0)
            removes += len(watch.first("address", "removed"))
            if run["facts"]["disturbed"]:
                disturbed += 1
            else:
                runs.append(run)
                stops.append(stop)
            print(f"round {index + 1}: run {run['facts']['total']:.3f} s, stop {stop['facts']['total']:.3f} s, "
                  f"addresses on the card after Stop {stop['facts']['windows_addresses_left']}", flush=True)
            time.sleep(gap)
        dns_used, end = cpu_seconds([dns])[dns] - dns_start, system_busy()
        watch.stop()
        events.stop()
        engine.json("POST", "/api/v1/runtime/stop")
        engine.quit()
    finally:
        if engine.process:
            engine.kill()
        engine.remove()
    return {"project": name, "rounds": rounds, "counted": len(runs), "refused": refused, "disturbed": disturbed,
            "machine_busy_percent": round(100 * (end[0] - busy[0]) / (end[1] - busy[1]), 1),
            "dns_client_cpu_seconds": round(dns_used, 3), "windows_address_adds": adds,
            "windows_address_removes": removes,
            "run_summary": summary.describe(runs, timeline.RUN_STEPS),
            "stop_summary": summary.describe(stops, timeline.STOP_STEPS), "runs": runs, "stops": stops}


def run_stop(engine, refused):
    """Run + Stop; a Run refused because another engine on this PC took an address off (today's
    code removes other copies' addresses) is retried and noted."""
    for _ in range(3):
        try:
            engine.json("POST", "/api/v1/runtime/start")
            break
        except RuntimeError as error:
            refused.append(str(error)[:200])
            time.sleep(1.0)
    engine.json("POST", "/api/v1/runtime/stop")
    time.sleep(0.5)
    return len(g.on_card())


def left_behind(server):
    """What stays on the card after Stop, after File › Close, after exit, after a kill + launch."""
    engine = g.LabInstance(server=server).start()
    result = {"refused": []}
    try:
        g.setup_small(engine)
        result["after_stop"] = run_stop(engine, result["refused"])
        engine.json("POST", "/api/v1/project/close")
        time.sleep(1.0)
        result["after_close_project"] = len(g.on_card())
        g.setup_small(engine)
        result["after_stop_2"] = run_stop(engine, result["refused"])
        engine.quit()
        result["after_exit"] = len(g.on_card())
        engine = g.LabInstance(server=server, scratch=engine.scratch).start()
        time.sleep(2.0)
        result["after_stop_3"] = run_stop(engine, result["refused"])
        engine.kill()
        result["after_kill"] = len(g.on_card())
        engine = g.LabInstance(server=server, scratch=engine.scratch).start()
        time.sleep(3.0)  # the launch clean-up runs in the background
        result["after_kill_and_launch"] = len(g.on_card())
        engine.quit()
    finally:
        if engine.process:
            engine.kill()
        engine.remove()
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--rounds", type=int, default=10)
    parser.add_argument("--server", choices=["base", "trial"], default="trial")
    parser.add_argument("--project", choices=sorted(projects.SOURCES), default="small")
    parser.add_argument("--skip-left", action="store_true", help="timing only")
    args = parser.parse_args()
    server = g.BASE if args.server == "base" else g.TRIAL
    result = {"server": args.server, "timing": timing(server, args.project, args.rounds)}
    if not args.skip_left:
        result["left_behind"] = left_behind(server)
    path = g.save(f"n10-{args.server}-{args.project}", result)
    timed = result["timing"]
    print(f"\nN10 {args.server} {args.project}: {timed['counted']} rounds counted ({timed['disturbed']} disturbed, "
          f"{len(timed['refused'])} refused), machine {timed['machine_busy_percent']} %, DNS Client "
          f"{timed['dns_client_cpu_seconds']} CPU s, Windows address adds {timed['windows_address_adds']} / removes "
          f"{timed['windows_address_removes']}\n\nRun\n{summary.markdown(timed['run_summary'])}\n\nStop\n"
          f"{summary.markdown(timed['stop_summary'])}\n\nleft behind {result.get('left_behind')}\nsaved {path}")


if __name__ == "__main__":
    main()
