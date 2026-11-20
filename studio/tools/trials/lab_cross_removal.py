"""Finding of the 2026-11-20 trials (change 0222): one ScadaBench copy takes another copy's
addresses off the Lab card at its Stop, Use, exit and launch.

    python studio/tools/trials/lab_cross_removal.py --rounds 10 --server base
    python studio/tools/trials/lab_cross_removal.py --rounds 10 --server trial

Two engines (copies) with their own data folders, both on the Lab card: B runs the 8-IED project
on 198.18.24.110-117 the whole time; A has the same project on 198.18.24.10-17. Each round A does
Run + Stop, then Use (the lab page's "Use the lab network"), then exits and is launched again.
After each action Windows' list of the Lab card is read: a fault is one of B's addresses gone.
`by_a` counts the ones A's own lab journal says A removed. At the end A is killed while running
and launched again: its own addresses must be gone, B's must stay. Result:
.local/trials/results/cross-*.json.
"""
import argparse
from pathlib import Path
import sys
import time

sys.path.insert(0, str(Path(__file__).resolve().parent))
import labgroup as g  # noqa: E402
import projects  # noqa: E402

B_FIRST = 110


def use_at(engine, first):
    project = engine.json("GET", "/api/v1/project")
    chosen = {ied["id"]: f"{g.RANGE}.{first + index}"
              for index, ied in enumerate(project["ieds"]) if ied.get("role") == "simulated"}
    engine.json("POST", "/api/v1/network/lab/use", {"addresses": chosen})
    return set(chosen.values())


def check(action, a, b_addresses, since):
    time.sleep(0.5)
    missing = sorted(b_addresses - g.on_card())
    by_a = sorted({ip for event, ip, _ in g.journal(a.data, since) if event == "removed" and ip in b_addresses})
    return {"action": action, "b_missing": missing, "by_a": by_a}


B_REFUSED = []  # B's own Runs refused meanwhile ("MMS bind failed": A took B's addresses off again)


def restore(b, b_addresses):
    """B Stop + Run puts its addresses back for the next action."""
    for _ in range(5):
        if not b_addresses - g.on_card():
            return
        b.json("POST", "/api/v1/runtime/stop")
        try:
            b.json("POST", "/api/v1/runtime/start")
        except RuntimeError as error:
            B_REFUSED.append(str(error)[:200])
            time.sleep(1.0)


def measure(rounds, server_name):
    server = g.BASE if server_name == "base" else g.TRIAL
    b = g.LabInstance(server=server).start()
    a = g.LabInstance(server=server).start()
    try:
        projects.open_in(b, projects.build("small"))
        b_addresses = use_at(b, B_FIRST)
        projects.open_in(a, projects.build("small"))
        a_addresses = use_at(a, 10)
        b.json("POST", "/api/v1/runtime/start")
        results = []
        for index in range(rounds):
            since = time.time()
            try:
                a.json("POST", "/api/v1/runtime/start")
            except RuntimeError as error:  # B's restore took A's addresses off (today)
                print(f"round {index + 1}: A's Run refused: {error}", flush=True)
            a.json("POST", "/api/v1/runtime/stop")
            results.append(check("A Run + Stop", a, b_addresses, since))
            restore(b, b_addresses)
            since = time.time()
            use_at(a, 10)
            results.append(check("A Use", a, b_addresses, since))
            restore(b, b_addresses)
            since = time.time()
            a.quit()
            results.append(check("A exit", a, b_addresses, since))
            restore(b, b_addresses)
            since = time.time()
            a = g.LabInstance(server=server, scratch=a.scratch).start()
            time.sleep(2.0)  # the launch clean-up runs in the background
            results.append(check("A launch", a, b_addresses, since))
            restore(b, b_addresses)
            faults = [r["action"] for r in results[-4:] if r["b_missing"]]
            print(f"round {index + 1}: B's addresses taken off by {faults or 'nothing'}", flush=True)
        restore(b, b_addresses)
        refused = ""
        try:
            a.json("POST", "/api/v1/runtime/start")
        except RuntimeError as error:  # today: B's restore Stop took A's new addresses off
            refused = str(error)[:300]
        a.kill()
        after_kill = sorted(a_addresses & g.on_card())
        a = g.LabInstance(server=server, scratch=a.scratch).start()
        time.sleep(2.0)
        kill = {"a_run_refused": refused, "a_left_after_kill": after_kill, "a_left_after_relaunch": sorted(a_addresses & g.on_card()),
                "b_missing_after_relaunch": sorted(b_addresses - g.on_card())}
        b.json("POST", "/api/v1/runtime/stop")
        a.quit()
        b.quit()
        left = sorted((a_addresses | b_addresses) & g.on_card())
    finally:
        for engine in (a, b):
            if engine.process:
                engine.kill()
            engine.remove()
    counted = [r for r in results if r["b_missing"]]
    return {"server": server_name, "rounds": rounds, "actions": len(results), "faults": len(counted),
            "faults_by_action": {name: sum(1 for r in counted if r["action"] == name)
                                 for name in ("A Run + Stop", "A Use", "A exit", "A launch")},
            "faults_removed_by_a": sum(1 for r in counted if r["by_a"]), "kill_relaunch": kill,
            "b_runs_refused": B_REFUSED,
            "left_after_both_exit": left, "results": results}


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--rounds", type=int, default=10)
    parser.add_argument("--server", choices=["base", "trial"], default="trial")
    args = parser.parse_args()
    result = measure(args.rounds, args.server)
    path = g.save(f"cross-{args.server}", result)
    print(f"\ncross-removal {args.server}: {result['faults']} of {result['actions']} actions took B's addresses off "
          f"({result['faults_removed_by_a']} by A's journal) {result['faults_by_action']}; B's Runs refused "
          f"{len(result['b_runs_refused'])}; kill+relaunch "
          f"{result['kill_relaunch']}; left after both exit {result['left_after_both_exit'] or 'nothing'}\nsaved {path}")


if __name__ == "__main__":
    main()
