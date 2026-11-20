"""Fix plan A12 and A13 live, through the engine's Modbus gateway (127.0.0.1:23503).

A13  A float at register 10 and a double point at register 11 share register 11: the app refuses
     them and says which points clash (when saving the list, or at the gateway start). The native
     check behind it is tested by tests/protocol/test_gateway_modbus_safety.py.
A12  Four masters connect and stay silent: a fifth is turned away while they are fresh, but after
     the 60 s idle limit the gateway has closed them and the fifth connects and reads. Forty short
     connections afterwards leave no threads behind in studio_gateway.exe.

Usage: python studio/tools/trials/fixplan_modbus.py
"""
import socket
import struct
import time

from fixplan_engine import (FixEngine, add_template, check, finish, place, rewrite_project, thread_count,
                            wait_for)

PORT, MODBUS_PORT, IDLE_S = 23060, 23503, 60
READ = bytes([3, 0, 10, 0, 2])  # FC3: the float at register 10


def ask(sock, pdu, transaction=1):
    sock.sendall(struct.pack(">HHHB", transaction, 0, len(pdu) + 1, 1) + pdu)
    header = sock.recv(7)
    if len(header) < 7:
        return None
    return sock.recv(struct.unpack(">H", header[4:6])[0] - 1)


def connect():
    return socket.create_connection(("127.0.0.1", MODBUS_PORT), timeout=5)


def turned_away(sock):
    """True when the gateway closes a new connection by itself (connection limit)."""
    try:
        return ask(sock, READ) is None
    except OSError:
        return True


def setup():
    engine = FixEngine(PORT).start()
    engine.wait_project()
    add_template(engine, "Bay")
    rewrite_project(engine, lambda document: place(document, {"Bay": ("127.67.23.20", 23104)}))
    bay = engine.project()["ieds"][0]["id"]
    engine.json("PUT", "/api/v1/tools/gateway/config", {"protocol": "modbus", "listen": "this_pc",
                                                         "real_commands": False,
                                                         "settings": {"port": MODBUS_PORT, "unit_id": 1}})
    engine.json("POST", "/api/v1/runtime/start")
    return engine, bay


def point(bay, kind, ref, address):
    return {"kind": kind, "ied": bay, "ref": ref, "address": address, "scale": 1.0, "description": ""}


def a13(engine, bay):
    overlapping = [point(bay, "float", "BayCTRL/GGIO1.AnIn1.mag.f", 10),
                   point(bay, "double", "BayCTRL/XCBR1.Pos.stVal", 11)]
    status, body, _ = engine.request("PUT", "/api/v1/tools/gateway/points", {"protocol": "modbus",
                                                                            "points": overlapping})
    where, text = "saving the point list", body.decode("utf-8", "replace")
    if status == 200:
        where = "gateway start"
        status, body, _ = engine.request("POST", "/api/v1/tools/gateway/start")
        text = body.decode("utf-8", "replace")
    # Both points named, or the second named and the first by its row (the point list's own words).
    named = "XCBR1" in text and ("AnIn1" in text or "point 1" in text)
    check("A13", "overlapping registers refused, naming the clashing points", status >= 400 and named,
          {"refused_at": where, "status": status, "message": text[:300]})
    engine.request("POST", "/api/v1/tools/gateway/stop")


def a12(engine, bay):
    engine.json("PUT", "/api/v1/tools/gateway/points", {"protocol": "modbus", "points": [
        point(bay, "float", "BayCTRL/GGIO1.AnIn1.mag.f", 10), point(bay, "double", "BayCTRL/XCBR1.Pos.stVal", 20)]})
    engine.json("POST", "/api/v1/tools/gateway/start")
    idle = [connect() for _ in range(4)]
    for sock in idle:
        ask(sock, READ)  # each is a real master that read once, then fell silent
    opened = time.perf_counter()
    fifth = connect()
    away = turned_away(fifth)
    fifth.close()
    time.sleep(max(0.0, IDLE_S + 3 - (time.perf_counter() - opened)))
    closed = sum(1 for sock in idle if turned_away(sock))
    with connect() as sock:
        answer = ask(sock, READ)
    check("A12", "a master still connects after four idle connections", away and closed == 4 and
          answer is not None and answer[:2] == bytes([3, 4]),
          {"fifth_turned_away_while_fresh": away, "idle_closed_after_s": IDLE_S + 3, "idle_closed": closed,
           "read_after": answer.hex() if answer else None})
    gateway = next(iter(engine.workers("studio_gateway.exe")))
    before = thread_count(gateway)
    for number in range(40):
        with connect() as sock:
            ask(sock, READ, number + 1)
    after = wait_for(lambda: thread_count(gateway) <= before + 2 and thread_count(gateway), 5) or thread_count(gateway)
    check("A12", "finished connection threads are joined", after <= before + 2,
          {"threads_before": before, "threads_after_40_connections": after})


def main():
    engine = None
    try:
        engine, bay = setup()
        a13(engine, bay)
        a12(engine, bay)
    finally:
        if engine:
            engine.close()
    return finish("fixplan_modbus")


if __name__ == "__main__":
    raise SystemExit(main())
