"""Fix plan A9 live: malformed GOOSE frames do not stop GOOSE.

Two simulated IEDs publish GOOSE inside ScadaBench (the corpus file goose_basic.scd, renamed).
studio_goose's bus is a UDP socket on 127.0.0.1 that every local program can reach, so the
check sends it the frames from the review (26-byte VLAN-tagged 0x88B8, truncated goosePdu
lengths) and thousands of random GOOSE-looking frames. GOOSE must keep working: the same
studio_goose process, and a value change still raises the publisher's stNum as received.

Usage: python studio/tools/trials/fixplan_goose.py [--fuzz 20000]
"""
import argparse
import os
import random
import socket
import struct
import subprocess
import time

from fixplan_engine import (STUDIO, Events, FixEngine, alive, check, finish, import_scl, place,
                            rewrite_project, wait_for)

PORT = 23050
SCD = STUDIO / "tests/parity/corpus/fixtures/goose_basic.scd"
DST, SRC = bytes.fromhex("010ccd010001"), bytes.fromhex("02aabbccddee")


def bus_port(pid):
    lines = subprocess.run(["netstat", "-ano", "-p", "UDP"], capture_output=True, text=True).stdout.splitlines()
    for line in lines:
        parts = line.split()
        if len(parts) >= 4 and parts[-1] == str(pid) and parts[1].startswith("127.0.0.1:"):
            return int(parts[1].rsplit(":", 1)[1])
    return None


def review_frames():
    tagged = DST + SRC + b"\x81\x00\x80\x00" + b"\x88\xb8" + struct.pack(">HHHH", 1, 8, 0, 0)  # 26 bytes, no PDU
    frames = [tagged, tagged[:20], DST + SRC + b"\x88\xb8"]
    header = DST + SRC + b"\x88\xb8" + struct.pack(">HHHH", 1, 200, 0, 0)
    for claim in (b"\x61\x81\xff", b"\x61\x82\xff\xff", b"\x61\x84\xff\xff\xff\xff", b"\x61\x89", b"\x61"):
        frames.append(header + claim + b"\x80\x05ab")  # goosePdu length past the end
    return frames


def fuzz_frames(count, seed=7):
    rng = random.Random(seed)
    for _ in range(count):
        prefix = DST + SRC + (b"\x81\x00\x80\x00" if rng.random() < 0.5 else b"") + b"\x88\xb8"
        yield prefix + os.urandom(rng.randint(0, 120))


def st_num(engine, name):
    watches = engine.json("GET", "/api/v1/goose")["watches"]
    found = [item for item in watches if item["publisher"]["name"] == name]
    return found[0]["st_num"] if found else None


def change(engine, ied, value):
    engine.json("POST", "/api/v1/signals/update", {"ied_id": ied, "ref": "GoosePubCTRL/XCBR1.Pos.stVal",
                                                    "value": value})


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--fuzz", type=int, default=20000)
    options = parser.parse_args()
    engine = FixEngine(PORT).start()
    events = None
    try:
        engine.wait_project()
        data = SCD.read_bytes()
        for name in ("GoosePub", "GooseSub"):
            import_scl(engine, f"{name}.scd", data.replace(b"Corpus", name.encode()), [name])
        rewrite_project(engine, lambda document: place(document, {"GoosePub": ("127.67.23.40", 23107),
                                                                  "GooseSub": ("127.67.23.41", 23108)}))
        publisher = next(ied["id"] for ied in engine.project()["ieds"] if ied["name"] == "GoosePub")
        events = Events(engine)
        engine.json("POST", "/api/v1/runtime/start")
        goose = wait_for(lambda: next(iter(engine.workers("studio_goose.exe")), None), 20)
        before = wait_for(lambda: st_num(engine, "GoosePub"), 20)
        change(engine, publisher, 2)
        first = wait_for(lambda: (st_num(engine, "GoosePub") or 0) > before and st_num(engine, "GoosePub"), 10)
        port = bus_port(goose)
        sender = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sent = 0
        for frame in review_frames():
            sender.sendto(frame, ("127.0.0.1", port))
            sent += 1
        for frame in fuzz_frames(options.fuzz):
            sender.sendto(frame, ("127.0.0.1", port))
            sent += 1
            if sent % 500 == 0:
                time.sleep(0.01)
        sender.close()
        time.sleep(1)
        change(engine, publisher, 1)
        after = wait_for(lambda: (st_num(engine, "GoosePub") or 0) > first and st_num(engine, "GoosePub"), 10)
        status = engine.json("GET", "/api/v1/goose")
        check("A9", "GOOSE keeps working after malformed frames",
              alive(goose) and goose in engine.workers("studio_goose.exe") and bool(after) and
              status["engine"]["running"] and not events.of("worker_failed"),
              {"bus_port": port, "frames_sent": sent, "st_num_before": before, "st_num_after_change": first,
               "st_num_after_fuzz_and_change": after, "studio_goose_same_process": goose in
               engine.workers("studio_goose.exe"), "engine_running": status["engine"]["running"],
               "state": status["state"]})
    finally:
        if events:
            events.close()
        engine.close()
    return finish("fixplan_goose")


if __name__ == "__main__":
    raise SystemExit(main())
