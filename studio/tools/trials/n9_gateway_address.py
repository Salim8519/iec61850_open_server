"""Trial N9 (speed and network trials 2026-11-20, change 0222): is the gateway's own Lab-card
address on the card only while the gateway runs?

    python studio/tools/trials/n9_gateway_address.py --cycles 10 --server base
    python studio/tools/trials/n9_gateway_address.py --cycles 10 --server trial

The gateway listens on its own address 198.18.24.50 (IEC 104, port 24404). `cycles` times: Start,
check that Windows lists the address and a TCP client connects to it, Stop, check Windows' list
again. Then Start once more and exit the app (Ctrl+Break): the address must be gone. Times are
the HTTP answers of Start and Stop. Result: .local/trials/results/n9-*.json.
"""
import argparse
from pathlib import Path
import socket
import statistics
import sys
import time

sys.path.insert(0, str(Path(__file__).resolve().parent))
import labgroup as g  # noqa: E402

OWN = f"{g.RANGE}.50"


def connects(ip, port, timeout=2.0):
    try:
        with socket.create_connection((ip, port), timeout=timeout):
            return True
    except OSError:
        return False


def measure(cycles, server_name):
    engine = g.LabInstance(server=g.BASE if server_name == "base" else g.TRIAL).start()
    try:
        g.setup_small(engine)
        g.ready_gateway(engine)
        engine.json("POST", "/api/v1/tools/gateway/network/own", {"address": OWN})
        after_add = OWN in g.on_card()
        rounds = []
        for index in range(cycles):
            clicked = time.perf_counter()
            try:
                engine.json("POST", "/api/v1/tools/gateway/start")
                started, error = round(time.perf_counter() - clicked, 3), ""
            except RuntimeError as failure:
                started, error = None, str(failure)[:300]
            on_while_running = OWN in g.on_card()
            reached = connects(OWN, g.GATEWAY_PORT) if not error else False
            clicked = time.perf_counter()
            engine.json("POST", "/api/v1/tools/gateway/stop")
            stopped = round(time.perf_counter() - clicked, 3)
            time.sleep(0.3)
            rounds.append({"start_s": started, "start_error": error, "on_while_running": on_while_running,
                           "client_connected": reached, "stop_s": stopped, "left_after_stop": OWN in g.on_card()})
            print(f"cycle {index + 1}: {rounds[-1]}", flush=True)
            time.sleep(0.5)
        engine.json("POST", "/api/v1/tools/gateway/start")
        engine.quit()
        left_after_exit = OWN in g.on_card()
    finally:
        if engine.process:
            engine.kill()
        engine.remove()
    starts = [r["start_s"] for r in rounds if r["start_s"] is not None]
    stops = [r["stop_s"] for r in rounds]
    return {"server": server_name, "cycles": cycles, "on_card_after_typing": after_add,
            "started": len(starts), "reached": sum(r["client_connected"] for r in rounds),
            "left_after_stop": sum(r["left_after_stop"] for r in rounds), "left_after_exit": left_after_exit,
            "start_median_s": statistics.median(starts) if starts else None, "start_worst_s": max(starts, default=None),
            "stop_median_s": statistics.median(stops), "stop_worst_s": max(stops), "rounds": rounds}


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--cycles", type=int, default=10)
    parser.add_argument("--server", choices=["base", "trial"], default="trial")
    args = parser.parse_args()
    result = measure(args.cycles, args.server)
    path = g.save(f"n9-{args.server}", result)
    print(f"\nN9 {args.server}: started {result['started']}/{args.cycles}, client reached {result['reached']}, "
          f"address left after Stop {result['left_after_stop']}/{args.cycles}, after exit {result['left_after_exit']}, "
          f"on the card right after typing {result['on_card_after_typing']}; Start median {result['start_median_s']} s "
          f"(worst {result['start_worst_s']}), Stop median {result['stop_median_s']} s (worst {result['stop_worst_s']})"
          f"\nsaved {path}")


if __name__ == "__main__":
    main()
