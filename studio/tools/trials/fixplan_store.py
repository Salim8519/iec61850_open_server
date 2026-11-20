"""Fix plan A3, A4, A5 live: the project file survives a killed engine, a damaged journal and a failed write.

A3  Diagram edits are sent back to back while the engine process is ended with TerminateProcess
    (taskkill /F) at a random moment, ROUNDS times. After each start the drawing must hold every
    edit the engine acknowledged (the in-flight one may or may not be there).
A4  The journal left by a killed engine is damaged in the middle; the next start must open the
    last autosave, move the journal into recovery\\ and say so, deleting nothing.
A5  With autosave.scadabench read-only, and again while another program holds it open without
    sharing, POST /project/save must answer an error that says the project file could not be
    written (not a generic server error); after the file is free again Save succeeds.

Usage: python studio/tools/trials/fixplan_store.py [--rounds 10]
"""
import argparse
import ctypes
import json
import os
import random
import stat
import threading
import time

from fixplan_engine import FixEngine, add_template, check, finish, scratch_dir

PORT = 23010


def node(index, x):
    return {"id": f"cb{index}", "type": "equipment", "position": {"x": x, "y": 40 * index},
            "data": {"label": f"CB {index}", "kind": "CB", "rotation": 0}}


def diagram(step):
    return {"nodes": [node(index, step * 10 + index) for index in range(3)], "edges": []}


def put(engine, step, revision):
    status, payload, _ = engine.request("PUT", "/api/v1/project/diagram", {
        "diagram": diagram(step), "revision": revision, "tab_id": "fixplan"}, timeout=10)
    return status, payload


def step_of(project):
    nodes = project["diagram"]["nodes"]
    return (nodes[0]["position"]["x"] // 10) if nodes else -1


def edit_until_killed(engine, delay):
    """Send edits until the engine dies; returns (last acknowledged step, last step sent)."""
    acked, sent, stop = [-1], [-1], threading.Event()
    revision = engine.project()["revision"]

    def editor():
        nonlocal revision
        step = 0
        while not stop.is_set():
            step += 1
            sent[0] = step
            try:
                status, payload = put(engine, step, revision)
            except OSError:
                return
            if status != 200:
                return
            revision = json.loads(payload)["revision"]
            acked[0] = step
    thread = threading.Thread(target=editor, daemon=True)
    thread.start()
    time.sleep(delay)
    engine.kill()
    stop.set()
    thread.join(15)
    return acked[0], sent[0]


def a3(data, rounds):
    for number in range(rounds):
        engine = FixEngine(PORT, data).start()
        try:
            engine.wait_project()
            if number == 0:
                add_template(engine, "StoreBay")
            delay = random.uniform(0.3, 2.0)
            acked, sent = edit_until_killed(engine, delay)
        finally:
            engine.close()
        engine = FixEngine(PORT, data).start()
        try:
            started = time.perf_counter()
            project = engine.wait_project()
            opened = round(time.perf_counter() - started, 3)
            found = step_of(project)
            ok = found in (acked, sent) and found >= acked and len(project["ieds"]) == 1
            check("A3", f"kill round {number + 1}", ok, {"killed_after_s": round(delay, 2), "acked_step": acked,
                  "sent_step": sent, "opened_step": found, "open_s": opened,
                  "recovery_notice": bool(project.get("startup_recovery"))})
        finally:
            engine.close()


def damage_journal(path):
    lines = path.read_bytes().split(b"\n")
    middle = max(1, (len(lines) - 1) // 2)
    lines[middle] = b'{"rev": "broken' + b"\x00" * 8
    path.write_bytes(b"\n".join(lines))
    return len(lines), middle


def a4(data):
    engine = FixEngine(PORT, data).start()
    try:
        engine.wait_project()
        engine.json("POST", "/api/v1/project/save")
        saved_step = step_of(engine.project())
        revision = engine.project()["revision"]
        for step in range(500, 506):
            status, payload = put(engine, step, revision)
            revision = json.loads(payload)["revision"]
        engine.kill()
    finally:
        engine.close()
    journal = data / "autosave.journal"
    count, broken = damage_journal(journal)
    before = sorted(os.listdir(data))
    engine = FixEngine(PORT, data).start()
    try:
        project = engine.wait_project()
        recovery = data / "recovery"
        moved = sorted(os.listdir(recovery)) if recovery.is_dir() else []
        notice = project.get("startup_recovery")
        ok = (not journal.exists() and any("journal" in name for name in moved) and
              (data / "autosave.scadabench").exists() and step_of(project) == saved_step and bool(notice))
        check("A4", "damaged journal moved to recovery", ok, {"journal_lines": count, "damaged_line": broken + 1,
              "files_before": before, "recovery": moved, "opened_step": step_of(project),
              "last_saved_step": saved_step, "notice": notice})
    finally:
        engine.close()


def hold_without_sharing(path):
    kernel = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel.CreateFileW.restype = ctypes.c_void_p
    handle = kernel.CreateFileW(str(path), 0x80000000, 0, None, 3, 0x80, None)  # GENERIC_READ, no share
    if handle in (None, ctypes.c_void_p(-1).value):
        raise ctypes.WinError(ctypes.get_last_error())
    return lambda: kernel.CloseHandle(ctypes.c_void_p(handle))


def a5(data):
    engine = FixEngine(PORT, data).start()
    try:
        engine.wait_project()
        target = data / "autosave.scadabench"
        for way in ("read-only", "held open by another program"):
            revision = engine.project()["revision"]
            put(engine, 700 if way == "read-only" else 800, revision)
            engine.json("POST", "/api/v1/project/save")  # the edit above is now on disk
            revision = engine.project()["revision"]
            put(engine, 701 if way == "read-only" else 801, revision)
            if way == "read-only":
                os.chmod(target, stat.S_IREAD)
                release = lambda: os.chmod(target, stat.S_IREAD | stat.S_IWRITE)  # noqa: E731
            else:
                release = hold_without_sharing(target)
            try:
                status, payload, _ = engine.request("POST", "/api/v1/project/save", timeout=30)
                again, _, _ = engine.request("POST", "/api/v1/project/save", timeout=30)
            finally:
                release()
            after, _, _ = engine.request("POST", "/api/v1/project/save", timeout=30)
            words = payload.decode("utf-8", "replace")
            ok = status == 503 and "could not write the project file" in words and again >= 400 and after == 200
            check("A5", f"save while autosave is {way}", ok, {"save_status": status,
                  "answer": payload[:200].decode("utf-8", "replace"), "second_save": again,
                  "save_after_release": after})
    finally:
        engine.close()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--rounds", type=int, default=10)
    options = parser.parse_args()
    data = scratch_dir("store") / "data"
    print("data folder:", data, flush=True)
    a3(data, options.rounds)
    a4(data)
    a5(data)
    return finish("fixplan_store")


if __name__ == "__main__":
    raise SystemExit(main())
