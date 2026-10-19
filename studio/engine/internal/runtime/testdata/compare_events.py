"""Compare the parity scenario's events of the Python and Go runtimes.

Usage: python compare_events.py PYTHON.json GO.json
Identical means (go-engine README): the same events per IED in the same order, with the same
kinds, fields and values, after normalising ids, timestamps, peers and correlation ids.
Left out, because they depend on wall-clock timing in both engines: generator `signal`
events, the worker's own buffered-report timer (`report_control` without a peer),
`automation_late` and the once-per-second `mms_access` summaries.
"""
import json
import sys
from collections import defaultdict

VOLATILE = {"peer": "<peer>", "correlation_id": "<id>", "observed_seconds": "<s>", "source_timestamp": "<ts>"}


def normalise(event):
    kind = event["kind"]
    if kind in {"automation_late", "mms_access"}:
        return None
    if kind == "report_control" and not event.get("peer"):
        return None
    if kind == "signal" and event.get("source") == "generator":
        return None
    result = {}
    for key, value in event.items():
        if key in {"id", "timestamp"}:
            continue
        if key in VOLATILE and value not in (0, None, ""):
            value = VOLATILE[key]
        if key == "updates" and isinstance(value, dict):
            value = {ref: ("<ts>" if ref.endswith(".t") and item else item) for ref, item in value.items()}
        if key == "value" and kind == "signal" and event.get("ref", "").endswith(".t") and value:
            value = "<ts>"
        result[key] = value
    if kind == "signal" and event.get("ref", "").endswith((".AnIn1.mag.f",)) and event.get("source") != "generator":
        result["value"] = "<generated>"
    return result


def unordered(item):
    """Python emits these from set iteration (hash order, random per process), so a run of
    them is compared as a multiset: initial snapshot signals and equipment states of one write."""
    if item["kind"] == "signal" and item.get("source") == "initial_native_snapshot":
        return "snapshot"
    return "state" if item["kind"] == "equipment_state" else None


def streams(events):
    grouped = defaultdict(list)
    for event in events:
        item = normalise(event)
        if item is not None:
            grouped[event.get("ied_id", "-")].append(item)
    result = {}
    for key, items in grouped.items():
        lines, run, run_kind = [], [], None
        for item in items + [{"kind": "<end>"}]:
            kind = unordered(item)
            if run and kind != run_kind:
                lines.extend(sorted(run))
                run = []
            if kind:
                run.append(json.dumps(item))
                run_kind = kind
            elif item["kind"] != "<end>":
                lines.append(json.dumps(item))
        result[key] = lines
    return result


def main():
    python = json.load(open(sys.argv[1], encoding="utf-8"))
    python = python["events"] if isinstance(python, dict) else python
    go = json.load(open(sys.argv[2], encoding="utf-8"))
    left, right = streams(python), streams(go)
    failures = 0
    for key in sorted(set(left) | set(right)):
        a, b = left.get(key, []), right.get(key, [])
        same = a == b
        print(f"stream {key}: python {len(a)} events, go {len(b)} events: {'identical' if same else 'DIFFERENT'}")
        if not same:
            failures += 1
            for index in range(max(len(a), len(b))):
                x = a[index] if index < len(a) else "<none>"
                y = b[index] if index < len(b) else "<none>"
                if x != y:
                    print(f"  first difference at {index}:\n    python {x}\n    go     {y}")
                    break
    sys.exit(1 if failures else 0)


if __name__ == "__main__":
    main()
