"""Write generated/journal.json: NetworkJournal bytes and replay results for the Go parity test.

Usage (from the worktree root):
  studio/.venv/Scripts/python.exe -I -B studio/engine/internal/netcfg/testdata/make_journal.py
The clock is fixed; every scenario records each step's result and the journal file bytes
(text mode, so CRLF lines on Windows).
"""
import sys
import tempfile
from datetime import datetime, timezone
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from vector_common import GATEWAY, b64, dumps, outcome, write  # noqa: E402
from studio_app.networking import journal as journal_module  # noqa: E402
from studio_app.networking.change_plan import gateway_plan  # noqa: E402
from studio_app.networking.journal import NetworkJournal, item_key  # noqa: E402

TIMES = [datetime(2026, 10, 20, 8, 15, 30, 123456, timezone.utc), datetime(2026, 10, 20, 8, 15, 31, 0, timezone.utc),
         datetime(2026, 1, 2, 3, 4, 5, 6, timezone.utc), datetime(2026, 12, 31, 23, 59, 59, 999999, timezone.utc)]


class FixedClock:
    """Stands in for journal.datetime: now() walks TIMES in order."""
    calls = 0

    @classmethod
    def now(cls, tz=None):
        value = TIMES[cls.calls % len(TIMES)]
        cls.calls += 1
        return value


OWNED = {"kind": "alias", "adapter_guid": "{1}", "adapter_index": 7, "adapter_name": "Lab NIC",
         "ip": "192.168.50.21", "prefix": 24, "owned": True}
FOREIGN = {**OWNED, "ip": "192.168.50.10", "owned": False}
RULE = {"kind": "firewall", "name": "SubstationStudio-MMS-0123456789ab", "owned": True}
BLOCK = {"name": r"TCP Query User{CD50883C-152F-4A3A-864C-00A52154808F}c:\studio\native\studio_gateway.exe",
         "display": "studio_gateway.exe"}
ALLOW, UNBLOCK = gateway_plan(GATEWAY, [], [BLOCK])["items"]
ODD = {"kind": "alias", "adapter_guid": "{2}", "ip": "10.0.0.5", "owned": 1, "adapter_name": "Ström \"Lab\" <&>",
       "zeta": [1.5, None, {"b": 2, "a": 1}], "prefix": None}

SCENARIOS = {
    "append_only": [["append", "applied", OWNED, "f1", ""], ["append", "removed", OWNED, "f2", "done"], ["outstanding"]],
    "owned_only": [["append", "applied", OWNED, "f1", ""], ["append", "applied", FOREIGN, "f1", ""],
                   ["append", "applied", RULE, "f1", ""], ["append", "failed", {**OWNED, "ip": "192.168.50.22"}, "f1", "x"],
                   ["outstanding"]],
    "leftovers": [["append", "applied", OWNED, "f1", ""], ["append", "applied", RULE, "f1", ""],
                  ["leftovers", ["alias"]], ["leftovers", []]],
    "gateway": [["append", "applied", ALLOW, "", ""], ["append", "applied", UNBLOCK, "", ""], ["outstanding"],
                ["append", "removed", UNBLOCK, "", ""], ["leftovers", ["gateway_firewall", "unblock"]]],
    "reorder": [["append", "applied", OWNED, "a", ""], ["append", "applied", RULE, "a", ""],
                ["append", "removed", OWNED, "b", ""], ["append", "applied", OWNED, "c", ""],
                ["append", "applied", RULE, "d", "again"], ["outstanding"]],
    "results": [["append", "skipped", ODD, "f", "é" * 600], ["append", "applied", ODD, "f", 5],
                ["append", "applied", ODD, "f", None], ["append", "failed", ODD, "f", {"x": [1, "y'z"]}],
                ["append", "bogus", ODD, "f", ""], ["outstanding"]],
}
RAW = {
    "blank_lines": b'\r\n{"event": "applied", "item": {"kind": "alias", "adapter_guid": "{1}", "ip": "1.2.3.4", '
                   b'"owned": true}}\r\n   \n\r{"event": "applied", "item": {"kind": "unblock", "name": "n", "owned": true}}',
    "corrupt": b'{"event": "applied", "item": {}}\n{not json\n',
    "missing_item": b'{"event": "applied"}\n',
    "not_owned": b'{"item": {"owned": false}}\n',
    "missing_event": b'{"item": {"owned": true, "kind": "firewall", "name": "r"}}\n',
}


def run(journal, step):
    if step[0] == "append":
        _, event, item, fingerprint, result = step
        return outcome(lambda: journal.append(event, item, fingerprint, result))
    if step[0] == "outstanding":
        return outcome(journal.outstanding)
    kinds = set(step[1])
    return outcome(lambda: journal.leftovers(lambda item: item["kind"] in kinds))


def main():
    journal_module.datetime = FixedClock
    scenarios = []
    for name, steps in SCENARIOS.items():
        FixedClock.calls = 0
        with tempfile.TemporaryDirectory() as folder:
            journal = NetworkJournal(Path(folder))
            results = [run(journal, step) for step in steps]
            scenarios.append({"name": name, "steps": steps, "results": results,
                              "file": b64(journal.path.read_bytes())})
    raws = []
    for name, data in RAW.items():
        with tempfile.TemporaryDirectory() as folder:
            journal = NetworkJournal(Path(folder))
            journal.path.parent.mkdir(parents=True)
            journal.path.write_bytes(data)
            raws.append({"name": name, "file": b64(data), "outstanding": outcome(journal.outstanding)})
    keys = [{"item": item, "key": outcome(lambda: item_key(item), str)}
            for item in (OWNED, RULE, ALLOW, UNBLOCK, ODD, {"kind": "x"}, {"name": "n"}, {"kind": "alias", "ip": 1})]
    write("journal.json", {"times": [t.isoformat() for t in TIMES], "scenarios": scenarios, "raw": raws,
                           "keys": keys, "dumps_check": dumps({"a": 1})})


if __name__ == "__main__":
    main()
