"""Trial N4 (speed and network trials 2026-11-20, change 0222): at launch, does the clean-up of
addresses an earlier session left on the Lab card race the gateway's (and the SNMP devices')
automatic start?

    python studio/tools/trials/n4_launch_order.py --launches 30 --server base
    python studio/tools/trials/n4_launch_order.py --launches 30 --server trial --exit kill

One data folder: the 8-IED project on the Lab card, the gateway on its own address 198.18.24.50
and one SNMP device at 198.18.24.60, both started (so both start again by themselves). Then
`launches` times: close the engine (--exit quit: the app's own exit; kill: End task), launch it
again on the same folder and wait until the gateway and the device say running or failed. A
fault is a start that failed or an address missing from Windows' list of the Lab card while its
device runs. Result: .local/trials/results/n4-*.json.
"""
import argparse
import json
from pathlib import Path
import sys
import time

sys.path.insert(0, str(Path(__file__).resolve().parent))
import labgroup as g  # noqa: E402

GATEWAY_IP = f"{g.RANGE}.50"
SNMP_IP = f"{g.RANGE}.60"


def settled(engine, timeout=30):
    """Gateway and SNMP device states once both stopped changing from "starting"/"stopped"."""
    deadline, gateway, device = time.time() + timeout, {}, {}
    while time.time() < deadline:
        gateway = engine.json("GET", "/api/v1/tools/gateway")
        device = (engine.json("GET", "/api/v1/tools/snmp").get("devices") or [{}])[0]
        if gateway.get("state") in ("running", "failed") and device.get("state") in ("running", "failed"):
            break
        time.sleep(0.1)
    return gateway, device


def journal_owned(engine, since):
    """Lab journal records after `since` with whose address it was ("for", empty for Release)."""
    from datetime import datetime
    path, found = Path(engine.data) / "network/lab_journal.jsonl", []
    for line in path.read_text(encoding="utf-8").splitlines() if path.is_file() else []:
        record = json.loads(line)
        at = datetime.fromisoformat(record["at"]).timestamp()
        if at >= since:
            found.append((record["event"], record["item"].get("ip", ""), at, record["item"].get("for", "")))
    return found


def one_launch(scratch, server, settle):
    engine = g.LabInstance(server=server, scratch=scratch)
    started = time.time()
    engine.start()
    gateway, device = settled(engine)
    time.sleep(settle)  # a clean-up still running would take an address off inside this
    card = g.on_card()
    records = g.journal(engine.data, started)
    since_launch = [(event, ip) for event, ip, _ in records if event == "removed"]
    outcome = {"seconds": round(time.time() - started, 2), "gateway": gateway.get("state"),
               "gateway_message": gateway.get("message", "")[:200], "snmp": device.get("state"),
               "snmp_error": (device.get("error") or "")[:200], "card": sorted(card),
               "removed_by_us": since_launch,
               "journal": [(event, ip, round(at - started, 3), owner) for event, ip, at, owner in journal_owned(engine, started)],
               "checked_at": round(time.time() - started, 3)}
    outcome["faults"] = [name for name, state, ip in (("gateway", outcome["gateway"], GATEWAY_IP),
                                                      ("snmp", outcome["snmp"], SNMP_IP))
                         if state != "running" or ip not in card]
    if outcome["gateway_message"].startswith("Address gone") and "gateway" not in outcome["faults"]:
        outcome["faults"].append("gateway")
    return engine, outcome


def measure(launches, server_name, exit_mode, settle):
    server = g.BASE if server_name == "base" else g.TRIAL
    first = g.LabInstance(server=server).start()
    scratch = first.scratch
    g.setup_small(first)
    g.ready_gateway(first)
    first.json("POST", "/api/v1/tools/gateway/network/own", {"address": GATEWAY_IP})
    first.json("POST", "/api/v1/tools/gateway/start")
    g.add_snmp_device(first, SNMP_IP)
    first.json("POST", "/api/v1/tools/snmp/start")
    time.sleep(1.5)  # autosave writes the project and both auto-start choices
    engine, rounds = first, []
    try:
        for index in range(launches):
            engine.quit() if exit_mode == "quit" else engine.kill()
            engine, outcome = one_launch(scratch, server, settle)
            rounds.append(outcome)
            print(f"launch {index + 1}: gateway {outcome['gateway']}, snmp {outcome['snmp']}, "
                  f"faults {outcome['faults'] or 'none'}" + (f" {outcome}" if outcome["faults"] else ""), flush=True)
        engine.json("POST", "/api/v1/tools/gateway/stop")
        engine.json("POST", "/api/v1/tools/snmp/stop")
        engine.quit()
    finally:
        if engine.process:
            engine.kill()
        left = sorted(g.on_card() & {GATEWAY_IP, SNMP_IP})
        engine.remove()
    faults = [r for r in rounds if r["faults"]]
    return {"server": server_name, "exit": exit_mode, "launches": launches, "settle": settle,
            "faulty_launches": len(faults), "gateway_faults": sum("gateway" in r["faults"] for r in rounds),
            "snmp_faults": sum("snmp" in r["faults"] for r in rounds), "left_after_last_exit": left,
            "rounds": rounds}


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--launches", type=int, default=30)
    parser.add_argument("--server", choices=["base", "trial"], default="trial")
    parser.add_argument("--exit", choices=["quit", "kill"], default="quit")
    parser.add_argument("--settle", type=float, default=1.0)
    args = parser.parse_args()
    result = measure(args.launches, args.server, args.exit, args.settle)
    path = g.save(f"n4-{args.server}-{args.exit}", result)
    print(f"\nN4 {args.server} ({args.exit}): {result['faulty_launches']} of {args.launches} launches with a fault "
          f"(gateway {result['gateway_faults']}, SNMP {result['snmp_faults']}); left after the last exit "
          f"{result['left_after_last_exit'] or 'nothing'}\nsaved {path}")


if __name__ == "__main__":
    main()
