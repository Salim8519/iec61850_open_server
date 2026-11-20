"""Trials N3 and N5 (speed and network trials 2026-11-20, change 0222): does a Release take an
address off the Lab card that another device of the same engine is just putting on?

    python studio/tools/trials/n3_n5_lock_races.py --idea n3 --mode add --loops 200 --server base
    python studio/tools/trials/n3_n5_lock_races.py --idea n3 --mode start --loops 200 --server trial
    python studio/tools/trials/n3_n5_lock_races.py --idea n5 --loops 200 --server trial

While one IED starts and stops in a loop (every Stop runs Release) and, for N3, SNMP Stop loops
too (it runs Release as well), the main loop
- n3 --mode add: chooses the gateway's own address 198.18.24.50, then .51, ... (today this puts
  it on the card first and saves the choice after);
- n3 --mode start: starts the gateway on its own address and stops it again (with ADR-0141 the
  address goes on the card at Start);
- n5: starts the SNMP device at 198.18.24.60 and stops it again.
A fault is the address missing from Windows' list of the Lab card (or the start refused) while
its device uses it. `by_us` says whether our own lab journal removed it (the race) or not
(another engine on this PC). Result: .local/trials/results/n3n5-*.json.
"""
import argparse
from pathlib import Path
import sys
import threading
import time

sys.path.insert(0, str(Path(__file__).resolve().parent))
import labgroup as g  # noqa: E402

OWN = [f"{g.RANGE}.{50 + n}" for n in range(4)]
SNMP_IP = f"{g.RANGE}.60"


class Disturber(threading.Thread):
    """Calls `action` again and again until stopped; counts calls and errors."""

    def __init__(self, action, pause):
        super().__init__(daemon=True)
        self.action, self.pause, self.running = action, pause, True
        self.calls, self.errors = 0, []

    def run(self):
        while self.running:
            try:
                self.action()
            except Exception as error:  # a refused start under load is not this trial's fault
                self.errors.append(str(error)[:200])
            self.calls += 1
            time.sleep(self.pause)


def removed_by_us(engine, ip, since):
    return any(event == "removed" and address == ip for event, address, _ in g.journal(engine.data, since))


def check(engine, ip, since, settle):
    time.sleep(settle)  # a Release that read the old state finishes inside this
    if ip in g.on_card():
        return None
    return {"ip": ip, "at": round(time.time() - since, 3), "by_us": removed_by_us(engine, ip, since)}


def n3_add(engine, loop, settle):
    ip, since = OWN[loop % len(OWN)], time.time()
    engine.json("POST", "/api/v1/tools/gateway/network/own", {"address": ip})
    return check(engine, ip, since, settle)


def n3_start(engine, loop, settle):
    ip, since = OWN[0], time.time()
    try:
        engine.json("POST", "/api/v1/tools/gateway/start")
    except RuntimeError as error:
        if ip in g.on_card():  # the address was there: the port of the last gateway, not this trial
            return {"ip": ip, "port_busy": str(error)[:200], "by_us": False, "not_counted": True}
        return {"ip": ip, "refused": str(error)[:300], "by_us": removed_by_us(engine, ip, since)}
    try:
        return check(engine, ip, since, settle)
    finally:
        engine.json("POST", "/api/v1/tools/gateway/stop")


def n5_start(engine, loop, settle):
    since = time.time()
    status = engine.json("POST", "/api/v1/tools/snmp/start")
    try:
        device = (status or {}).get("devices", [{}])[0]
        if device.get("state") != "running":
            return {"ip": SNMP_IP, "refused": device.get("error", "")[:300],
                    "by_us": removed_by_us(engine, SNMP_IP, since)}
        return check(engine, SNMP_IP, since, settle)
    finally:
        engine.json("POST", "/api/v1/tools/snmp/stop")


def measure(idea, mode, loops, server, settle):
    engine = g.LabInstance(server=g.BASE if server == "base" else g.TRIAL).start()
    try:
        ieds = list(g.setup_small(engine))
        g.ready_gateway(engine)
        g.add_snmp_device(engine, SNMP_IP)
        if idea == "n3" and mode == "start":
            engine.json("POST", "/api/v1/tools/gateway/network/own", {"address": OWN[0]})
        ied = ieds[0]
        disturbers = [Disturber(lambda: (engine.json("POST", f"/api/v1/runtime/ieds/{ied}/start"),
                                         engine.json("POST", f"/api/v1/runtime/ieds/{ied}/stop")), 0.0)]
        if idea == "n3":
            disturbers.append(Disturber(lambda: engine.json("POST", "/api/v1/tools/snmp/stop"), 0.01))
        for item in disturbers:
            item.start()
        step = {"n3": n3_add if mode == "add" else n3_start, "n5": n5_start}[idea]
        faults, skipped, started = [], [], time.time()
        for loop in range(loops):
            fault = step(engine, loop, settle)
            if fault and fault.get("not_counted"):
                skipped.append(fault)
                fault = None
            if fault:
                fault["loop"] = loop
                faults.append(fault)
                print(f"loop {loop + 1}: FAULT {fault}", flush=True)
            elif (loop + 1) % 20 == 0:
                print(f"loop {loop + 1}: {len(faults)} faults so far", flush=True)
        for item in disturbers:
            item.running = False
        for item in disturbers:
            item.join(30)
        engine.json("POST", "/api/v1/runtime/stop")
        engine.quit()  # the app's exit takes the session's addresses off (a kill would leave them)
        return {"idea": idea, "mode": mode, "server": server, "loops": loops, "settle": settle,
                "seconds": round(time.time() - started, 1), "faults": faults,
                "faults_by_us": sum(1 for f in faults if f.get("by_us")), "not_counted": skipped,
                "disturber_calls": [d.calls for d in disturbers],
                "disturber_errors": [d.errors[:5] for d in disturbers]}
    finally:
        if engine.process:
            engine.kill()
        engine.remove()


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--idea", choices=["n3", "n5"], required=True)
    parser.add_argument("--mode", choices=["add", "start"], default="add", help="n3 only")
    parser.add_argument("--loops", type=int, default=200)
    parser.add_argument("--server", choices=["base", "trial"], default="trial")
    parser.add_argument("--settle", type=float, default=0.25)
    args = parser.parse_args()
    result = measure(args.idea, args.mode, args.loops, args.server, args.settle)
    path = g.save(f"n3n5-{args.idea}-{args.mode}-{args.server}", result)
    print(f"\n{args.idea} {args.mode} {args.server}: {len(result['faults'])} faults in {args.loops} loops "
          f"({result['faults_by_us']} removed by this engine), disturber calls {result['disturber_calls']}, "
          f"{result['seconds']} s\nsaved {path}")


if __name__ == "__main__":
    main()
