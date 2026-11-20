"""Trial R7: SCADA master Connect to an IEC 104 outstation with many points, today against a trial.

    python studio/tools/trials/r7_master_connect.py --points 5000 --rounds 10 ^
        --base studio/build/engine/studio_server_base.exe=<native dir> --trial <exe>=<native dir>

The outstation (IEC 104, common address 1, 127.67.26.30:26404, --points points, started here
outside any engine) is `--outstation gateway`: a separate studio_gateway (single, double, float,
integer points), or `standin`: r7_station.py, which never answers the counter interrogation. Each
session starts an engine instance (temp data folder) in the SCADA master role and does
Connect / wait for every point / Disconnect rounds. Timed: the Connect answer, the moment
GET /tools/master shows every point, and Disconnect (clicked once every point is there).
Writes .local/trials/results/r7-<time>.json.
"""
import argparse
import json
from pathlib import Path
import statistics
import sys
import time

sys.path.insert(0, str(Path(__file__).resolve().parent))
from fixplan_engine import wait_for  # noqa: E402
from instance import Instance  # noqa: E402
from runstop_ab import side  # noqa: E402
from r7_station import SilentCounterStation  # noqa: E402
from step0_run_stop import RESULTS  # noqa: E402

STUDIO = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(STUDIO / "backend"))
from studio_app.runtime.worker_process import WorkerProcess  # noqa: E402

HOST, PORT = "127.67.26.30", 26404
KINDS = [("single", 0.4), ("double", 0.2), ("float", 0.3), ("integer", 0.1)]


def outstation(native, count):
    points, values, address = [], [], 1
    for kind, share in KINDS:
        for index in range(int(count * share)):
            point = f"{kind}{index}"
            points.append({"id": point, "kind": kind, "address": address})
            value = {"single": True, "double": 2, "float": 230.5 + index, "integer": index}[kind]
            values.append({"id": point, "value": value, "quality": 0})
            address += 1
    process = WorkerProcess(Path(native) / "studio_gateway.exe", lambda event: None)
    process.request("start", protocol="iec104", settings={"bind": HOST, "port": PORT, "common_address": 1},
                    points=points)
    process.request("update", values=values)
    return process, len(points)


def points_now(engine):
    return engine.json("GET", "/api/v1/tools/master")["points"]


def one_round(engine, count):
    began = time.perf_counter()
    status = engine.json("POST", "/api/v1/tools/master/connect", timeout=120)
    answered = time.perf_counter() - began
    complete = wait_for(lambda: points_now(engine) >= count, 90, pause=0.02)
    every = time.perf_counter() - began
    result = {"connect": round(answered, 3), "all_points": round(every, 3), "points": points_now(engine),
              "state": status["state"], "complete": bool(complete)}
    time.sleep(0.5)
    result["message"] = engine.json("GET", "/api/v1/tools/master")["message"]
    began = time.perf_counter()
    engine.json("POST", "/api/v1/tools/master/disconnect", timeout=120)
    result["disconnect"] = round(time.perf_counter() - began, 3)
    return result


def session(config, count, rounds, gap, into):
    with Instance(server=config["server"], native=config["native"], shared=True) as engine:
        wait_for(lambda: engine.request("GET", "/api/v1/project")[0] == 200, 60, pause=0.2)
        engine.json("POST", "/api/v1/tools/gateway/role", {"role": "master"})
        engine.json("PUT", "/api/v1/tools/master/config", {"protocol": "iec104", "host": HOST,
                                                            "settings": {"port": PORT, "common_address": 1}})
        for _ in range(rounds):
            into.append(one_round(engine, count))
            print(json.dumps(into[-1]), flush=True)
            time.sleep(gap)


def summary(values):
    return {"median": round(statistics.median(values), 3), "worst": round(max(values), 3)} if values else None


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--base", type=side, required=True)
    parser.add_argument("--trial", type=side, required=True)
    parser.add_argument("--points", type=int, default=5000)
    parser.add_argument("--rounds", type=int, default=10)
    parser.add_argument("--per-session", type=int, default=5)
    parser.add_argument("--gap", type=float, default=1.0)
    parser.add_argument("--outstation", choices=["gateway", "standin"], default="gateway")
    args = parser.parse_args()
    if args.outstation == "gateway":
        gateway, count = outstation(args.base["native"], args.points)
    else:
        gateway, count = SilentCounterStation(HOST, PORT, args.points), args.points
    data = {"base": [], "trial": []}
    try:
        while any(len(rounds) < args.rounds for rounds in data.values()):
            for name in ("base", "trial"):
                wanted = min(args.per_session, args.rounds - len(data[name]))
                if wanted > 0:
                    print(name, flush=True)
                    session(getattr(args, name), count, wanted, args.gap, data[name])
    finally:
        gateway.close()
    result = {"points": count, "outstation": args.outstation, "base": args.base, "trial": args.trial, "rounds": data,
              "summary": {name: {key: summary([r[key] for r in rounds]) for key in ("connect", "all_points", "disconnect")}
                          | {"incomplete": sum(not r["complete"] for r in rounds)} for name, rounds in data.items()}}
    RESULTS.mkdir(parents=True, exist_ok=True)
    path = RESULTS / f"r7-{args.outstation}-{time.strftime('%Y%m%d-%H%M%S')}.json"
    path.write_text(json.dumps(result, indent=1), encoding="utf-8")
    print(json.dumps(result["summary"], indent=1), f"\nsaved {path}")


if __name__ == "__main__":
    main()
