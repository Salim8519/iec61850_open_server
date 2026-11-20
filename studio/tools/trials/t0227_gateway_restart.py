"""Change 0227: does a gateway Start right after a Stop find its port still taken?

    python studio/tools/trials/t0227_gateway_restart.py --cycles 200
    python studio/tools/trials/t0227_gateway_restart.py --cycles 200 --client --disturb

For IEC 104 and Modbus TCP, on the gateway's own Lab address 198.18.27.50 and on 127.0.0.1
(This PC), the station gateway starts and stops `--cycles` times; each Start follows the last
Stop at once. `--client` connects a TCP client before each Stop (the gateway closes the
connection first, as with a real SCADA). `--disturb` starts and stops one IED (on the Lab card at
198.18.27.10) in a loop meanwhile, as trial N3 did. Windows is the witness: right after every
Stop answer the script reads Windows' TCP listener table and asks whether the old
studio_gateway process still exists. Ports 27000-27999, Lab addresses 198.18.27.0/24 (this
change's range). Result: .local/trials/results/t0227-*.json.
"""
import argparse
import ctypes
from pathlib import Path
import socket
import sys
import threading
import time

sys.path.insert(0, str(Path(__file__).resolve().parent))
import instance  # noqa: E402
import labgroup as g  # noqa: E402
import projects  # noqa: E402
import winlisten  # noqa: E402

MAIN = Path(r"C:\Users\Salim\Desktop\Windsurf\iec61850_open_server")  # worktrees lack ignored folders
RANGE = "198.18.27"
OWN = RANGE + ".50"
PORTS = {"iec104": 27404, "modbus": 27502}
SYNCHRONIZE = 0x00100000


def alive(pid):
    """True while Windows still has a process object for pid that has not exited."""
    handle = ctypes.windll.kernel32.OpenProcess(SYNCHRONIZE, False, pid)
    if not handle:
        return False
    try:
        return ctypes.windll.kernel32.WaitForSingleObject(handle, 0) == 0x102  # WAIT_TIMEOUT
    finally:
        ctypes.windll.kernel32.CloseHandle(handle)


def configure(engine, protocol, listen):
    config = engine.json("GET", "/api/v1/tools/gateway/config")
    settings = dict((config.get("settings") or {}).get(protocol) or {}, port=PORTS[protocol])
    engine.json("PUT", "/api/v1/tools/gateway/config", {"protocol": protocol, "settings": settings})
    proposal = engine.json("POST", "/api/v1/tools/gateway/propose", {"protocol": protocol})
    engine.json("PUT", "/api/v1/tools/gateway/points", {"protocol": protocol, "points": proposal["points"][:20]})
    if listen == "own":
        engine.json("POST", "/api/v1/tools/gateway/network/own", {"address": OWN})
    else:
        engine.json("PUT", "/api/v1/tools/gateway/network", {"choice": "this_pc", "allowed": ""})
    return OWN if listen == "own" else "127.0.0.1"


class Disturber(threading.Thread):
    def __init__(self, engine, ied):
        super().__init__(daemon=True)
        self.engine, self.ied, self.running, self.calls, self.errors = engine, ied, True, 0, []

    def run(self):
        while self.running:
            try:
                self.engine.json("POST", f"/api/v1/runtime/ieds/{self.ied}/start")
                self.engine.json("POST", f"/api/v1/runtime/ieds/{self.ied}/stop")
            except Exception as error:
                self.errors.append(str(error)[:200])
            self.calls += 1


def cycle(engine, bind, port, client):
    """One Start, optional client, Stop; what Windows shows right after the Stop answer."""
    record, started = {}, time.perf_counter()
    try:
        engine.json("POST", "/api/v1/tools/gateway/start")
    except RuntimeError as error:
        record["refused"] = str(error)[:300]
        record["listeners"] = {f"{ip}:{p}": pid for (ip, p), pid in winlisten.listeners().items() if p == port}
        return record
    record["start_s"] = round(time.perf_counter() - started, 3)
    pid = winlisten.listeners(bind).get((bind, port))
    connection = None
    if client:
        try:
            connection = socket.create_connection((bind, port), timeout=2)
        except OSError as error:
            record["client"] = str(error)[:120]
    stopped = time.perf_counter()
    try:
        engine.json("POST", "/api/v1/tools/gateway/stop")
    except RuntimeError as error:  # the engine still ends the program; the next Start goes on
        record["stop_failed"] = str(error)[:200]
    record["stop_s"] = round(time.perf_counter() - stopped, 3)
    still = winlisten.listeners(bind).get((bind, port))
    record["after_stop"] = {"listener": still, "old_alive": bool(pid and alive(pid))}
    if connection:
        connection.close()
    return record


def run_combo(engine, protocol, listen, cycles, client):
    bind = configure(engine, protocol, listen)
    port, faults, records = PORTS[protocol], [], []
    for number in range(cycles):
        record = cycle(engine, bind, port, client)
        records.append(record)
        if "refused" in record or "stop_failed" in record or record["after_stop"]["listener"] \
                or record["after_stop"]["old_alive"]:
            record["cycle"] = number
            faults.append(record)
            print(f"  {protocol} {listen} cycle {number + 1}: {record}", flush=True)
    starts = sorted(r["start_s"] for r in records if "start_s" in r)
    stops = sorted(r["stop_s"] for r in records if "stop_s" in r)
    summary = {"protocol": protocol, "listen": listen, "bind": bind, "port": port, "cycles": cycles,
               "refused": sum(1 for r in records if "refused" in r),
               "stop_failed": sum(1 for r in records if "stop_failed" in r),
               "listener_after_stop": sum(1 for r in records if (r.get("after_stop") or {}).get("listener")),
               "old_alive_after_stop": sum(1 for r in records if (r.get("after_stop") or {}).get("old_alive")),
               "start_median_s": starts[len(starts) // 2] if starts else None, "start_worst_s": starts[-1] if starts else None,
               "stop_median_s": stops[len(stops) // 2] if stops else None, "stop_worst_s": stops[-1] if stops else None}
    print(f"{protocol} {listen}: {summary['refused']} refused, {summary['stop_failed']} Stop failed, "
          f"{summary['listener_after_stop']} listener left, "
          f"{summary['old_alive_after_stop']} old process alive after Stop; Start {summary['start_median_s']} s "
          f"median ({summary['start_worst_s']}), Stop {summary['stop_median_s']} s ({summary['stop_worst_s']})", flush=True)
    return summary, faults


def measure(args):
    projects.CIDS = MAIN / "test_cids"
    projects.SOURCES = {"small": [(projects.CIDS / "substation/DUQM-panels.scadabench", None)]}
    projects.RANGE, projects.IED_PORT = RANGE, 27102
    engine = instance.Instance(port=instance.free_port(27000, 27999), native=Path(args.native),
                               server=Path(args.server), shared=True)
    engine.start()
    disturber = None
    try:
        projects.open_in(engine, projects.build("small"))
        engine.json("POST", "/api/v1/tools/gateway/role", {"role": "gateway"})
        if args.disturb:
            ieds = list(projects.use_lab_card(engine))
            disturber = Disturber(engine, ieds[0])
            disturber.start()
        results, faults = [], []
        for protocol in args.protocols.split(","):
            for listen in args.listens.split(","):
                summary, found = run_combo(engine, protocol, listen, args.cycles, args.client)
                results.append(summary)
                faults += found
        if disturber:
            disturber.running = False
            disturber.join(30)
        engine.json("POST", "/api/v1/runtime/stop")
        return {"server": args.server, "native": args.native, "cycles": args.cycles, "client": args.client,
                "disturb": args.disturb, "results": results, "faults": faults[:60],
                "disturber": {"calls": disturber.calls, "errors": disturber.errors[:5]} if disturber else None}
    finally:
        engine.stop()


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--cycles", type=int, default=200)
    parser.add_argument("--protocols", default="iec104,modbus")
    parser.add_argument("--listens", default="own,this_pc")
    parser.add_argument("--client", action="store_true")
    parser.add_argument("--disturb", action="store_true")
    parser.add_argument("--server", default=str(projects.REPO / "studio/build/engine/studio_server.exe"))
    parser.add_argument("--native", default=str(MAIN / "studio/native/build/Release"))
    parser.add_argument("--label", default="today")
    args = parser.parse_args()
    result = measure(args)
    path = g.save(f"t0227-{args.label}", result)
    print(f"saved {path}")


if __name__ == "__main__":
    main()
