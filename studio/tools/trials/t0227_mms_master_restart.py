"""Change 0227: the same "start right after stop" check for the IEDs' MMS ports and the SCADA master.

    python studio/tools/trials/t0227_mms_master_restart.py --cycles 200

- IED: one IED on the Lab card (198.18.27.10:27102) starts and stops `--cycles` times, each start
  right after the last stop; then the whole Run starts and stops `--runs` times (8 IEDs,
  198.18.27.10-17). A fault is a refused start ("MMS bind failed" or any other error).
- Master: a second engine runs the station gateway (IEC 104, then Modbus TCP) on its own Lab
  address 198.18.27.50:27404/27502; this engine's SCADA master connects and disconnects
  `--cycles` times from All networks and from its own address 198.18.27.51. A fault is a refused
  Connect or Disconnect.
Ports 27000-27999, Lab addresses 198.18.27.0/24. Result: .local/trials/results/t0227-mms-master-*.json.
"""
import argparse
from pathlib import Path
import sys
import time

sys.path.insert(0, str(Path(__file__).resolve().parent))
import instance  # noqa: E402
import labgroup as g  # noqa: E402
import projects  # noqa: E402
import t0227_gateway_restart as t  # noqa: E402


def timed(engine, method, path, body=None):
    started = time.perf_counter()
    try:
        engine.json(method, path, body)
        return round(time.perf_counter() - started, 3), None
    except RuntimeError as error:
        return round(time.perf_counter() - started, 3), str(error)[:240]


def loop(name, cycles, start, stop):
    faults, starts = [], []
    for number in range(cycles):
        seconds, error = start()
        starts.append(seconds)
        if error:
            faults.append({"cycle": number, "start": error})
            print(f"  {name} cycle {number + 1}: {error}", flush=True)
        seconds, error = stop()
        if error:
            faults.append({"cycle": number, "stop": error})
            print(f"  {name} cycle {number + 1} stop: {error}", flush=True)
    starts.sort()
    summary = {"name": name, "cycles": cycles, "faults": len(faults), "start_median_s": starts[len(starts) // 2],
               "start_worst_s": starts[-1]}
    print(f"{name}: {len(faults)} faults in {cycles}; start {summary['start_median_s']} s median "
          f"({summary['start_worst_s']})", flush=True)
    return summary, faults


def engine_at(args):
    return instance.Instance(port=instance.free_port(27000, 27999), native=Path(args.native),
                             server=Path(args.server), shared=True).start()


def ieds(args):
    engine = engine_at(args)
    try:
        projects.open_in(engine, projects.build("small"))
        chosen = list(projects.use_lab_card(engine))
        one = loop("IED start/stop", args.cycles,
                   lambda: timed(engine, "POST", f"/api/v1/runtime/ieds/{chosen[0]}/start"),
                   lambda: timed(engine, "POST", f"/api/v1/runtime/ieds/{chosen[0]}/stop"))
        run = loop("Run/Stop (8 IEDs)", args.runs, lambda: timed(engine, "POST", "/api/v1/runtime/start"),
                   lambda: timed(engine, "POST", "/api/v1/runtime/stop"))
        return [one, run]
    finally:
        engine.stop()


def masters(args):
    gateway, master = engine_at(args), engine_at(args)
    results = []
    try:
        projects.open_in(gateway, projects.build("small"))
        gateway.json("POST", "/api/v1/tools/gateway/role", {"role": "gateway"})
        master.json("POST", "/api/v1/tools/gateway/role", {"role": "master"})
        for protocol in ("iec104", "modbus"):
            t.configure(gateway, protocol, "own")
            gateway.json("POST", "/api/v1/tools/gateway/start")
            config = master.json("GET", "/api/v1/tools/master/config")
            settings = dict((config.get("settings") or {}).get(protocol) or {}, port=t.PORTS[protocol])
            master.json("PUT", "/api/v1/tools/master/config", {"protocol": protocol, "host": t.OWN, "settings": settings})
            for local in ("all", "own"):
                if local == "own":
                    master.json("POST", "/api/v1/tools/master/network/own", {"address": t.RANGE + ".51"})
                else:
                    master.json("PUT", "/api/v1/tools/master/network", {"local": ""})
                results.append(loop(f"master {protocol} from {local}", args.cycles,
                                    lambda: timed(master, "POST", "/api/v1/tools/master/connect"),
                                    lambda: timed(master, "POST", "/api/v1/tools/master/disconnect")))
            gateway.json("POST", "/api/v1/tools/gateway/stop")
        return results
    finally:
        master.stop()
        gateway.stop()


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--cycles", type=int, default=200)
    parser.add_argument("--runs", type=int, default=50)
    parser.add_argument("--parts", default="ieds,masters")
    parser.add_argument("--server", default=str(projects.REPO / "studio/build/engine/studio_server.exe"))
    parser.add_argument("--native", default=str(t.MAIN / "studio/native/build/Release"))
    parser.add_argument("--label", default="today")
    args = parser.parse_args()
    projects.CIDS = t.MAIN / "test_cids"
    projects.SOURCES = {"small": [(projects.CIDS / "substation/DUQM-panels.scadabench", None)]}
    projects.RANGE, projects.IED_PORT = t.RANGE, 27102
    found = []
    if "ieds" in args.parts:
        found += ieds(args)
    if "masters" in args.parts:
        found += masters(args)
    result = {"cycles": args.cycles, "runs": args.runs, "results": [summary for summary, _ in found],
              "faults": [fault for _, faults in found for fault in faults][:80]}
    print(f"saved {g.save(f't0227-mms-master-{args.label}', result)}")


if __name__ == "__main__":
    main()
