"""Query compact append-only Studio run logs without modifying them."""
import argparse
import json
from collections import deque
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1] / "data" / "logs"


def event_files(root, run_id=None):
    runs = [root / run_id] if run_id else sorted(path for path in root.iterdir() if path.is_dir())
    for run in runs:
        if run.is_dir():
            yield from sorted(run.glob("events-*.jsonl"))


def selected_files(root, args):
    if not args.lifecycle:
        return event_files(root, args.run)
    target = root / args.run / "manifest.jsonl" if args.run else root / "runs.jsonl"
    return iter([target] if target.is_file() else [])


def matches(item, args):
    if args.kind and item.get("kind") != args.kind:
        return False
    if args.ied and item.get("ied_id") != args.ied:
        return False
    if args.ref and args.ref not in (item.get("ref"), item.get("control_ref"), item.get("status_ref")) \
            and not (item.get("kind") == "signal_summary" and args.ref in item.get("values", {})):  # ADR-0115
        return False
    if args.text and args.text.casefold() not in json.dumps(item).casefold():
        return False
    return True


def query(root, args):
    found = deque(maxlen=args.limit)
    for path in selected_files(root, args):
        for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
            try:
                item = json.loads(line)
            except ValueError as error:
                raise ValueError(f"{path}:{number}: invalid JSON: {error}") from error
            if matches(item, args):
                found.append(item)
    return list(found)


def parser():
    result = argparse.ArgumentParser(description=__doc__)
    result.add_argument("--root", type=Path, default=ROOT)
    result.add_argument("--run")
    result.add_argument("--kind")
    result.add_argument("--ied")
    result.add_argument("--ref")
    result.add_argument("--text")
    result.add_argument("--lifecycle", action="store_true",
                        help="query append-only run lifecycle indexes instead of events")
    result.add_argument("--limit", type=int, default=50)
    return result


def main():
    args = parser().parse_args()
    if not 1 <= args.limit <= 1000:
        raise SystemExit("--limit must be between 1 and 1000")
    if not args.root.exists():
        return 0
    for item in query(args.root, args):
        print(json.dumps(item, separators=(",", ":"), ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
