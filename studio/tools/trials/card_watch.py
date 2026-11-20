"""Owner check helper: log every address change on one card and the DNS Client CPU, per second.

    python studio/tools/trials/card_watch.py --card Ethernet --seconds 300

For real cards, which a trial cannot change at night (UAC, cable). Start this, then click Run
and Stop in the app (wait 30 s after each). It reads only Windows: the card's address list
(with Windows' creation time of each address) and the Dnscache svchost's CPU. Nothing is
changed. Writes the log to .local/trials/results/card-watch-<time>.json as well.
"""
import argparse
import json
from pathlib import Path
import sys
import time

sys.path.insert(0, str(Path(__file__).resolve().parent))
import winaddr  # noqa: E402
from winclock import now  # noqa: E402
from winproc import cpu_seconds, service_pid  # noqa: E402

RESULTS = Path(__file__).resolve().parents[3] / ".local/trials/results"


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--card", default="Ethernet", help="the card's name as Windows shows it")
    parser.add_argument("--seconds", type=float, default=300.0)
    args = parser.parse_args()
    card = winaddr.card_index(args.card)
    if card is None:
        raise SystemExit(f"Windows has no card named {args.card!r}")
    dns = service_pid("Dnscache")
    start, records = now(), []
    known = winaddr.addresses(interface=card)
    cpu, second = cpu_seconds([dns])[dns], int(start)
    print(f"watching {args.card} (index {card}); {len(known)} addresses now; Ctrl+C to end early")
    try:
        while now() - start < args.seconds:
            time.sleep(0.01)
            moment, current = now(), winaddr.addresses(interface=card)
            for ip in current.keys() - known.keys():
                records.append({"t": round(moment - start, 3), "added": ip, "state": current[ip].state})
                print(f"{moment - start:8.3f} s  + {ip} ({current[ip].state})", flush=True)
            for ip in known.keys() - current.keys():
                records.append({"t": round(moment - start, 3), "removed": ip})
                print(f"{moment - start:8.3f} s  - {ip}", flush=True)
            known = current
            if int(moment) != second:
                value = cpu_seconds([dns])[dns]
                records.append({"t": round(moment - start, 3), "dns_cores": round(value - cpu, 3)})
                if value - cpu > 0.2:
                    print(f"{moment - start:8.3f} s  DNS Client {value - cpu:.2f} cores", flush=True)
                cpu, second = value, int(moment)
    except KeyboardInterrupt:
        pass
    RESULTS.mkdir(parents=True, exist_ok=True)
    out = RESULTS / f"card-watch-{time.strftime('%Y%m%d-%H%M%S')}.json"
    out.write_text(json.dumps({"card": args.card, "records": records}, indent=1), encoding="utf-8")
    print(f"saved {out}")


if __name__ == "__main__":
    main()
