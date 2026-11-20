"""Trials R4 and R6: Stop with a real-IED session open, today's code against a trial build.

    python studio/tools/trials/real_stop_ab.py --case r4 --rounds 10 ^
        --base studio/build/engine/studio_server_base.exe=<native dir> --trial <exe>=<native dir>

The "real device" is a separate studio_ied (template ReportTest) on 127.67.26.10:26102 started
here, outside any engine. Each session starts a station engine (temp data folder, loopback
addresses, no Lab card) with ReportTest as a real IED (report blocks used) and six simulated
template IEDs on 127.67.26.21-26, then does Run, wait, Stop rounds.
  r4: the device answers. Stop is timed once the session is live and holds its report block;
      an independent studio_live client checks the block is held while running and free after.
  r6: the real IED's address is 127.67.26.11:26102, where a listener accepts and never answers
      (MMS connect hangs); Stop is clicked 1 s after Run, while the session is connecting.
Writes .local/trials/results/real-<case>-<time>.json and prints medians and worst.
"""
import argparse
import json
from pathlib import Path
import socket
import statistics
import sys
import tempfile
import threading
import time

sys.path.insert(0, str(Path(__file__).resolve().parent))
from fixplan_engine import add_template, place, rewrite_project, wait_for  # noqa: E402
from instance import Instance  # noqa: E402
from runstop_ab import side  # noqa: E402
from sse import EventRecorder  # noqa: E402
from step0_run_stop import RESULTS  # noqa: E402
from winclock import now  # noqa: E402

STUDIO = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(STUDIO / "backend"))
from studio_app.runtime.worker_process import WorkerProcess  # noqa: E402
from studio_app.scl.importer import import_scl  # noqa: E402
from studio_app.scl.templates import make_template  # noqa: E402

DEVICE, SILENT, PORT = "127.67.26.10", "127.67.26.11", 26102
RCB = "ReportTestCTRL/LLN0.RP.EventsRCB"
BAYS = {f"Bay{n}": (f"127.67.26.{20 + n}", PORT) for n in range(1, 7)}


class Device:
    """studio_ied with the ReportTest model on DEVICE:PORT, and an independent studio_live client."""

    def __init__(self, native):
        self.native = Path(native)
        model = import_scl(make_template("ReportTest"))[0]
        config = Path(tempfile.mkdtemp(prefix="r4-device-")) / "device.cfg"
        config.write_text(model["cfg"], encoding="utf-8")
        self.ied = WorkerProcess(self.native / "studio_ied.exe", lambda event: None)
        self.ied.request("start", cfg=str(config), ip=DEVICE, port=PORT, controls=model["controls"], edition=2)

    def block(self):
        client = WorkerProcess(self.native / "studio_live.exe", lambda event: None)
        try:
            client.request("connect", timeout=15, ip=DEVICE, port=PORT)
            return client.request("rcb_status", timeout=30, rcbs=[RCB])["blocks"][0]
        finally:
            client.close()

    def close(self):
        self.ied.close()


def silent_listener():
    """Accepts MMS connections and never answers (a device that does not answer)."""
    server = socket.socket()
    server.bind((SILENT, PORT))
    server.listen(16)
    held = []
    threading.Thread(target=lambda: [held.append(server.accept()) for _ in iter(int, 1)], daemon=True).start()
    return server


def station(config, case):
    engine = Instance(server=config["server"], native=config["native"], shared=True).start()
    wait_for(lambda: engine.request("GET", "/api/v1/project")[0] == 200, 60, pause=0.2)
    for name in ["ReportTest", *BAYS]:
        add_template(engine, name)
    target = DEVICE if case == "r4" else SILENT
    rewrite_project(engine, lambda document: place(document, BAYS))
    real = next(ied["id"] for ied in engine.json("GET", "/api/v1/project")["ieds"] if ied["name"] == "ReportTest")
    engine.json("PUT", f"/api/v1/ieds/{real}/settings", {"role": "real", "live": {
        "ip": target, "port": PORT, "access": "auto_reports", "poll_ms": 1000}})
    return engine


def session_state(engine):
    sessions = engine.json("GET", "/api/v1/live")["sessions"]
    return sessions[0]["state"] if sessions else None


def one_round(engine, events, device, case):
    engine.json("POST", "/api/v1/runtime/start")
    if case == "r4":
        if not wait_for(lambda: session_state(engine) == "live", 30):
            return {"error": f"session not live: {session_state(engine)}"}
        time.sleep(1.0)
        held = device.block()
    else:
        time.sleep(1.0)
        held = {"state": session_state(engine)}
    click = now()
    engine.json("POST", "/api/v1/runtime/stop", timeout=120)
    answered = now()
    time.sleep(0.5)
    stopped = [e["timestamp"] for e in events.since(click) if e.get("kind") == "runtime" and e.get("state") == "stopped"]
    result = {"stop": round(answered - click, 4), "stopped_event": round(min(stopped) - click, 4) if stopped else None,
              "while_running": held}
    if case == "r4":
        result["after_stop"] = device.block()
    return result


def summary(values):
    return {"median": round(statistics.median(values), 3), "worst": round(max(values), 3)} if values else None


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--case", choices=["r4", "r6"], required=True)
    parser.add_argument("--base", type=side, required=True)
    parser.add_argument("--trial", type=side, required=True)
    parser.add_argument("--rounds", type=int, default=10)
    parser.add_argument("--per-session", type=int, default=5)
    parser.add_argument("--gap", type=float, default=2.0)
    args = parser.parse_args()
    device = Device(args.base["native"])
    listener = silent_listener() if args.case == "r6" else None
    data = {"base": [], "trial": []}
    try:
        while any(len(rounds) < args.rounds for rounds in data.values()):
            for name in ("base", "trial"):
                engine = station(getattr(args, name), args.case)
                events = EventRecorder(engine).start()
                try:
                    for _ in range(min(args.per_session, args.rounds - len(data[name]))):
                        data[name].append(one_round(engine, events, device, args.case))
                        print(name, json.dumps(data[name][-1]), flush=True)
                        time.sleep(args.gap)
                finally:
                    events.stop()
                    engine.stop()
    finally:
        device.close()
        if listener:
            listener.close()
    result = {"case": args.case, "base": args.base, "trial": args.trial, "rounds": data,
              "summary": {name: {"stop": summary([r["stop"] for r in rounds if "stop" in r]),
                                 "stopped_event": summary([r["stopped_event"] for r in rounds if r.get("stopped_event")]),
                                 "errors": sum("error" in r for r in rounds)} for name, rounds in data.items()}}
    RESULTS.mkdir(parents=True, exist_ok=True)
    path = RESULTS / f"real-{args.case}-{time.strftime('%Y%m%d-%H%M%S')}.json"
    path.write_text(json.dumps(result, indent=1), encoding="utf-8")
    print(json.dumps(result["summary"], indent=1), f"\nsaved {path}")


if __name__ == "__main__":
    main()
