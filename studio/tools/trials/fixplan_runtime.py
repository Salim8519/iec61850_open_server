"""Fix plan A19 and A20 live: adding an IED to a running station fails, then succeeds.

Bay1 runs alone. Bay2's address 127.67.23.31:23106 is held by another program, so its start
fails: Bay2 must not be listed as running and no extra studio_ied.exe may stay (A19); with the
address free again the same Start works, never "IED already started" (A20). ROUNDS times.
The engine's own address check catches the busy address before a worker starts, so a last round
freezes Bay2's new worker the moment it appears: its start then fails after the worker started,
and the worker must be stopped and forgotten.

Usage: python studio/tools/trials/fixplan_runtime.py [--rounds 5]
"""
import argparse
import socket
import threading
import time

from fixplan_engine import (FixEngine, add_template, alive, check, finish, place, rewrite_project, suspend,
                            wait_for)

PORT = 23040
BAY1, BAY2 = ("127.67.23.30", 23105), ("127.67.23.31", 23106)


def start(engine, ied):
    began = time.perf_counter()
    status, body, _ = engine.request("POST", f"/api/v1/runtime/ieds/{ied}/start", timeout=60)
    return status, body.decode("utf-8", "replace"), round(time.perf_counter() - began, 2)


def stop(engine, ied):
    return engine.request("POST", f"/api/v1/runtime/ieds/{ied}/stop", timeout=60)[0]


def round_trip(engine, ids, number):
    holder = socket.socket()
    holder.bind(BAY2)
    holder.listen(1)
    try:
        failed = start(engine, ids["Bay2"])
        time.sleep(1)
        running = engine.project()["running_ieds"]
        workers = len(engine.workers("studio_ied.exe"))
    finally:
        holder.close()
    check("A19", f"failed add is stopped and forgotten (round {number})",
          failed[0] >= 400 and ids["Bay2"] not in running and ids["Bay1"] in running and workers == 1,
          {"status": failed[0], "answer": failed[1][:220], "running": [name for name, ied in ids.items()
                                                                         if ied in running],
           "studio_ied_processes": workers})
    again = start(engine, ids["Bay2"])
    running = engine.project()["running_ieds"]
    workers = wait_for(lambda: len(engine.workers("studio_ied.exe")) == 2 and 2, 5) or \
        len(engine.workers("studio_ied.exe"))
    check("A20", f"the same Start works once the address is free (round {number})",
          again[0] == 200 and ids["Bay2"] in running and "already started" not in again[1] and workers == 2,
          {"status": again[0], "answer": again[1][:160], "seconds": again[2], "studio_ied_processes": workers})
    stop(engine, ids["Bay2"])


def frozen_start(engine, ids):
    known = set(engine.workers("studio_ied.exe"))
    answer = {}
    thread = threading.Thread(target=lambda: answer.update(result=start(engine, ids["Bay2"])))
    thread.start()
    fresh = None
    while thread.is_alive() and fresh is None:
        fresh = next((pid for pid in engine.workers("studio_ied.exe") if pid not in known), None)
        time.sleep(0.002)
    if fresh:
        suspend(fresh)
    thread.join(120)
    status, body, seconds = answer.get("result", (0, "", 0))
    running = engine.project()["running_ieds"]
    left = wait_for(lambda: len(engine.workers("studio_ied.exe")) == 1 and 1, 10) or \
        len(engine.workers("studio_ied.exe"))
    check("A19", "worker frozen during its start: stopped and forgotten",
          fresh is not None and status >= 400 and ids["Bay2"] not in running and left == 1 and not alive(fresh),
          {"frozen_pid": fresh, "status": status, "answer": body[:200], "seconds": seconds,
           "running": [name for name, ied in ids.items() if ied in running], "studio_ied_processes": left,
           "frozen_alive": bool(fresh) and alive(fresh)})
    again = start(engine, ids["Bay2"])
    check("A20", "Start after the frozen start works", again[0] == 200 and "already started" not in again[1],
          {"status": again[0], "answer": again[1][:160]})
    stop(engine, ids["Bay2"])


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--rounds", type=int, default=5)
    options = parser.parse_args()
    engine = FixEngine(PORT).start()
    try:
        engine.wait_project()
        add_template(engine, "Bay1")
        add_template(engine, "Bay2")
        rewrite_project(engine, lambda document: place(document, {"Bay1": BAY1, "Bay2": BAY2}))
        ids = {ied["name"]: ied["id"] for ied in engine.project()["ieds"]}
        status, body, _ = start(engine, ids["Bay1"])
        if status != 200:
            raise RuntimeError(f"Bay1 did not start: {body}")
        for number in range(1, options.rounds + 1):
            round_trip(engine, ids, number)
        frozen_start(engine, ids)
    finally:
        engine.close()
    return finish("fixplan_runtime")


if __name__ == "__main__":
    raise SystemExit(main())
