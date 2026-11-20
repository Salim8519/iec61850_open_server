"""Fix plan A7, A8, A10, A11, A14, A18 live: a real IED that stops answering, and frozen workers.

Two engines: B simulates the device "Field" at 127.67.23.10:23102; A has "Field" as a real IED
(Allow commands on) and a Modbus gateway on 127.0.0.1:23502 whose coils 0 and 1 operate it.
A program "stops answering" by being suspended (NtSuspendProcess, like Task Manager).

A8   studio_control frozen: a Devices operate ends after 30 s with "result unknown"; the frozen
     program is gone; a second operate of the same reference meanwhile is refused at once.
A7   Four SCADA commands, the first stuck in the frozen studio_control: gateway Stop is quick
     and the three waiting ones are logged "refused: the gateway stopped before this command ran".
A14  The Modbus master waiting for that first command gets its answer when Stop runs.
A11  studio_gateway frozen: Stop ends with the gateway "stopped" and its program gone.
A18  B's IED worker frozen: A marks the real IED's values stale (live_stale).
A10  B's frozen IED worker (its pipe full of updates) does not hang B's Stop or Quit.

Usage: python studio/tools/trials/fixplan_realied.py
"""
import socket
import struct
import threading
import time

from fixplan_engine import (Events, FixEngine, add_template, alive, check, finish, place, resume,
                            rewrite_project, suspend, wait_for)

DEVICE, GATEWAY_PORT = ("127.67.23.10", 23102), 23502
# Deadlines from the code with one frozen worker: a request already in flight to it ends after its
# 10 s timeout, then Close sends "stop" (3 s), waits 3 s for the exit and kills it; Quit adds the
# 2 s HTTP shutdown.
STOP_LIMIT, QUIT_LIMIT = 16.0, 18.0
HALT_LIMIT = 20.0  # a frozen gateway program: 10 s halt request, then the same 3 s + 3 s close


def modbus(pdu, transaction=1, timeout=40):
    """One Modbus TCP request on a new connection: (answer PDU or error text, seconds)."""
    started = time.perf_counter()
    try:
        with socket.create_connection(("127.0.0.1", GATEWAY_PORT), timeout=timeout) as sock:
            sock.sendall(struct.pack(">HHHB", transaction, 0, len(pdu) + 1, 1) + pdu)
            header = sock.recv(7)
            if len(header) < 7:
                return "closed", round(time.perf_counter() - started, 2)
            body = sock.recv(struct.unpack(">H", header[4:6])[0] - 1)
            return body.hex(), round(time.perf_counter() - started, 2)
    except OSError as error:
        return type(error).__name__, round(time.perf_counter() - started, 2)


def coil(address, on):
    return bytes([5, 0, address, 0xFF if on else 0, 0])


def device(port):
    engine = FixEngine(port).start()
    engine.wait_project()
    add_template(engine, "Field")
    rewrite_project(engine, lambda document: place(document, {"Field": DEVICE}))
    field = engine.project()["ieds"][0]["id"]
    engine.json("POST", "/api/v1/signals/source", {"ied_id": field, "ref": "FieldCTRL/GGIO1.AnIn1.mag.f", "spec": {
        "mode": "ramp", "offset": 100, "amplitude": 50, "period": 1, "seed": 1, "overridden": False}})
    engine.json("POST", "/api/v1/runtime/start")
    return engine


def station(port):
    engine = FixEngine(port).start()
    engine.wait_project()
    add_template(engine, "Field")
    field = engine.project()["ieds"][0]["id"]
    engine.json("PUT", f"/api/v1/ieds/{field}/settings", {"role": "real", "live": {
        "ip": DEVICE[0], "port": DEVICE[1], "allow_commands": True}})
    engine.json("PUT", "/api/v1/tools/gateway/config", {"protocol": "modbus", "listen": "this_pc", "real_commands": True,
                                                         "settings": {"port": GATEWAY_PORT, "unit_id": 1}})
    points = [{"kind": "command_double", "ied": field, "ref": f"FieldCTRL/{name}.Pos", "address": index,
               "scale": 1.0, "description": f"Field {name}"} for index, name in enumerate(["XCBR1", "XSWI1"])]
    engine.json("PUT", "/api/v1/tools/gateway/points", {"protocol": "modbus", "points": points})
    engine.json("POST", "/api/v1/runtime/start")
    wait_for(lambda: engine.json("GET", "/api/v1/live")["sessions"][0]["state"] == "live", 20)
    return engine, field


def operate(engine, field, value):
    started = time.perf_counter()
    status, body, _ = engine.request("POST", f"/api/v1/live/{field}/operate",
                                     {"ref": "FieldCTRL/XCBR1.Pos", "value": value, "confirm": True}, timeout=90)
    return status, body.decode("utf-8", "replace"), round(time.perf_counter() - started, 2)


def control_pid(engine, known=()):
    return wait_for(lambda: next((pid for pid in engine.workers("studio_control.exe") if pid not in known), None), 10)


def a8(engine, field, events):
    status, body, _ = operate(engine, field, True)  # starts the warm studio_control
    frozen = control_pid(engine)
    suspend(frozen)
    first = {}
    thread = threading.Thread(target=lambda: first.update(answer=operate(engine, field, False)))
    thread.start()
    time.sleep(1)
    second = operate(engine, field, False)
    thread.join(60)
    unknown = events.of("real_command_unknown")
    check("A8", "second operate while one runs is refused", second[0] == 400 and "still running" in second[1],
          {"status": second[0], "answer": second[1][:160], "seconds": second[2]})
    answer = first.get("answer", (0, "", 0))
    check("A8", "frozen control program: 'result unknown', program gone",
          answer[0] == 503 and "result unknown" in answer[1] and bool(unknown) and not alive(frozen),
          {"warm_operate": status, "status": answer[0], "answer": answer[1][:200], "seconds": answer[2],
           "event": unknown[-1].get("reason") if unknown else None, "frozen_pid_alive": alive(frozen)})
    status, body, seconds = operate(engine, field, True)
    check("A8", "next operate works with a fresh control program", status == 200 and '"ok":true' in body,
          {"status": status, "seconds": seconds})
    return frozen


def a7_a14(engine, frozen_before):
    engine.json("POST", "/api/v1/tools/gateway/start")
    warm = modbus(coil(0, False))
    frozen = control_pid(engine, known={frozen_before}) or control_pid(engine)
    suspend(frozen)
    answers = {}
    threads = [threading.Thread(target=lambda n=n, p=p: answers.update({n: modbus(p, n)}))
               for n, p in [(1, coil(0, True)), (2, coil(1, True)), (3, coil(1, False)), (4, coil(0, False))]]
    for thread in threads:
        thread.start()
        time.sleep(0.3)
    time.sleep(1.5)
    started = time.perf_counter()
    status, body, _ = engine.request("POST", "/api/v1/tools/gateway/stop", timeout=60)
    stop_s = round(time.perf_counter() - started, 2)
    for thread in threads:
        thread.join(45)
    log = engine.json("GET", "/api/v1/tools/gateway/commands?after=0")["entries"]
    refused = [entry for entry in log if entry.get("applied") == "refused: the gateway stopped before this command ran"]
    state = engine.json("GET", "/api/v1/tools/gateway")["state"]
    check("A7", "Stop refuses the waiting commands, quickly", status == 200 and stop_s < 3 and len(refused) == 3
          and state == "stopped", {"warm_command": warm, "stop_status": status, "stop_s": stop_s,
                                   "refused": len(refused), "state": state})
    check("A14", "the master waiting on the stuck command is answered at Stop", answers.get(1, ("", 99))[1] < 10,
          {"answers": answers})
    time.sleep(32)  # the stuck command times out in studio_control (30 s)
    log = engine.json("GET", "/api/v1/tools/gateway/commands?after=0")["entries"]
    stuck = [entry for entry in log if entry.get("label", "").startswith("FieldCTRL/XCBR1.Pos ON") and
             "refused" not in entry.get("applied", "")]
    check("A7", "the stuck command's log line after its timeout", not alive(frozen) and bool(stuck) and
          "unknown" in stuck[-1].get("applied", ""), {"frozen_pid_alive": alive(frozen), "entry": stuck[-1:]})


def a11(engine):
    engine.json("POST", "/api/v1/tools/gateway/start")
    gateway = wait_for(lambda: next(iter(engine.workers("studio_gateway.exe")), None), 10)
    suspend(gateway)
    started = time.perf_counter()
    status, body, _ = engine.request("POST", "/api/v1/tools/gateway/stop", timeout=60)
    seconds = round(time.perf_counter() - started, 2)
    state = engine.json("GET", "/api/v1/tools/gateway")
    check("A11", "frozen gateway program: Stop leaves 'stopped', program gone",
          state["state"] == "stopped" and not alive(gateway) and seconds < HALT_LIMIT,
          {"stop_status": status, "answer": body[:200].decode("utf-8", "replace"), "seconds": seconds,
           "state": state["state"], "message": state.get("message"), "gateway_alive": alive(gateway)})


def a18_a10(device_engine, engine, events):
    worker = next(iter(device_engine.workers("studio_ied.exe")))
    since = time.perf_counter()
    suspend(worker)
    stale = wait_for(lambda: events.of("live_stale", since), 40, 0.5)
    check("A18", "device stops answering: values marked stale", bool(stale), {
        "seconds": round(stale[0]["_at"] - since, 1) if stale else None,
        "message": stale[0].get("message") if stale else None})
    time.sleep(3)  # updates keep coming for the frozen worker: its pipe fills up
    started = time.perf_counter()
    status, _, _ = device_engine.request("POST", "/api/v1/runtime/stop", timeout=120)
    stop_s = round(time.perf_counter() - started, 2)
    check("A10", "Stop with a frozen IED worker returns in time", status == 200 and stop_s < STOP_LIMIT and
          not alive(worker), {"status": status, "stop_s": stop_s, "limit_s": STOP_LIMIT, "worker_alive": alive(worker)})
    device_engine.json("POST", "/api/v1/runtime/start")
    worker = wait_for(lambda: next(iter(device_engine.workers("studio_ied.exe")), None), 20)
    suspend(worker)
    time.sleep(3)
    quit_s = device_engine.quit(60)
    left = {pid: name for pid, name in device_engine.workers().items() if name.startswith("studio_")}
    check("A10", "Quit with a frozen IED worker returns in time", quit_s is not None and quit_s < QUIT_LIMIT and
          not left, {"quit_s": quit_s, "limit_s": QUIT_LIMIT, "left_running": left})


def main():
    device_engine = engine = events = None
    try:
        device_engine = device(23031)
        engine, field = station(23030)
        events = Events(engine)
        frozen = a8(engine, field, events)
        a7_a14(engine, frozen)
        a11(engine)
        a18_a10(device_engine, engine, events)
    finally:
        for item in (events,):
            if item:
                item.close()
        for item in (engine, device_engine):
            if item:
                for pid in item.workers():
                    try:
                        resume(pid)
                    except OSError:
                        pass
                item.close()
    return finish("fixplan_realied")


if __name__ == "__main__":
    raise SystemExit(main())
