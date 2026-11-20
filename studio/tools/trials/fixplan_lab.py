"""Fix plan A6 live, on the ScadaBench Lab card (no administrator prompt).

An IED is put on the Lab card at 198.18.23.50 (POST /network/lab/use). Another address,
198.18.23.99, is added through the network service directly, the way another program or
another ScadaBench would; it is not in this app's journal. The check records which addresses
the card holds while running, after Stop, after a new launch and after Quit, and that Clean up
(preview only: applying it may need administrator rights) lists nothing it did not add.

Real cards (an approved address set as the card's own address in Windows) need administrator
rights: that part is an Owner check. Usage: python studio/tools/trials/fixplan_lab.py
"""
import json
import time

from fixplan_engine import FixEngine, add_template, check, finish, scratch_dir, wait_for

# The app refuses loopback (127.x) on the Lab card, so the check uses the RFC 2544 test range.
PORT, MINE, OTHER = 23070, "198.18.23.50", "198.18.23.99"
PIPE = r"\\.\pipe\ScadaBenchLabNet"


def open_pipe(timeout=10):
    """The service answers one client at a time; a busy pipe is tried again (as WaitNamedPipe)."""
    deadline = time.time() + timeout
    while True:
        try:
            return open(PIPE, "r+b", buffering=0)
        except OSError:
            if time.time() > deadline:
                raise
            time.sleep(0.05)


def service(request):
    with open_pipe() as pipe:
        pipe.write(json.dumps(request).encode() + b"\n")
        data = b""
        while not data.endswith(b"\n"):
            chunk = pipe.read(65536)
            if not chunk:
                break
            data += chunk
    return json.loads(data)


def on_card():
    return sorted(address for address in service({"op": "status"}).get("addresses", [])
                  if address.startswith("198.18.23."))


def main():
    data = scratch_dir("lab") / "data"
    engine = FixEngine(PORT, data).start()
    try:
        engine.wait_project()
        add_template(engine, "LabBay")
        bay = engine.project()["ieds"][0]["id"]
        lab = engine.json("GET", "/api/v1/network/lab")
        if not lab.get("ready"):
            raise RuntimeError(f"Lab card not ready: {lab}")
        engine.json("POST", "/api/v1/network/lab/use", {"addresses": {bay: MINE}})
        added = service({"op": "add", "ip": OTHER})
        engine.json("POST", "/api/v1/runtime/start")
        running = wait_for(lambda: MINE in on_card() and on_card(), 15) or on_card()
        engine.json("POST", "/api/v1/runtime/stop")
        stopped = wait_for(lambda: MINE not in on_card() and on_card() or None, 10) or on_card()
        check("A6", "Stop takes the IED's own Lab address off", MINE in running and MINE not in stopped,
              {"other_added_by_service": added.get("ok"), "while_running": running, "after_stop": stopped})
        other_after_stop = OTHER in stopped
        preview = engine.json("GET", "/api/v1/network/cleanup/preview")
        listed = json.dumps(preview)
        check("A6", "Clean up lists nothing ScadaBench did not add", OTHER not in listed and MINE not in listed,
              {"cleanup_items": preview.get("items", preview)})
        if OTHER not in on_card():
            service({"op": "add", "ip": OTHER})
        engine.json("POST", "/api/v1/runtime/start")
        wait_for(lambda: MINE in on_card(), 15)
        quit_s = engine.quit(60)
        after_quit = on_card()
        engine = FixEngine(PORT, data).start()
        engine.wait_project()
        time.sleep(2)  # the launch takes off what an earlier session left (ADR-0127)
        after_launch = on_card()
        check("A6", "Quit takes the IED's own Lab address off", MINE not in after_quit,
              {"quit_s": quit_s, "after_quit": after_quit, "after_next_launch": after_launch})
        check("A6", "an address another program put on the Lab card: kept?",
              other_after_stop and OTHER in after_quit and OTHER in after_launch,
              {"after_stop": other_after_stop, "after_quit": OTHER in after_quit,
               "after_next_launch": OTHER in after_launch,
               "rule": "release.go releaseLab: every address no running IED of this app uses comes off (ADR-0127); "
                       "fix plan Part B 'not decided': copies running together remove each other's addresses"},
              owner=True)
    finally:
        engine.close()
        for address in on_card():
            if address in (MINE, OTHER):
                service({"op": "remove", "ip": address})
    return finish("fixplan_lab")


if __name__ == "__main__":
    raise SystemExit(main())
