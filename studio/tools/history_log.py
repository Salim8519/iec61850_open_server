"""Append, verify, and query the immutable engineering-change ledger."""
import argparse
import hashlib
import json
import subprocess
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
HISTORY = ROOT / "docs" / "history"
GENESIS = "0" * 64
MAX_SEGMENT_LINES = 200
FIELDS = {"id", "at", "intent", "reason", "files", "expected", "evidence",
          "tags", "prev", "hash"}


def ledger_paths(directory=HISTORY):
    return sorted(directory.glob("ledger-[0-9][0-9][0-9][0-9].jsonl"))


def read_records(directory=HISTORY):
    for path in ledger_paths(directory):
        for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
            if line.strip():
                yield path, number, json.loads(line)


def canonical(record):
    unsigned = {key: value for key, value in record.items() if key != "hash"}
    return json.dumps(unsigned, sort_keys=True, ensure_ascii=False,
                      separators=(",", ":")).encode("utf-8")


def record_hash(record):
    return hashlib.sha256(canonical(record)).hexdigest()


def validate_record(record, previous, seen):
    errors = []
    if not isinstance(record, dict) or set(record) != FIELDS:
        return [f"fields must be exactly {sorted(FIELDS)}"]
    if not isinstance(record["id"], str) or not record["id"] or record["id"] in seen:
        errors.append("id must be a unique non-empty string")
    try:
        datetime.fromisoformat(record["at"].replace("Z", "+00:00"))
    except (TypeError, ValueError):
        errors.append("at must be an ISO-8601 timestamp")
    for key in ("intent", "reason"):
        if not isinstance(record[key], str) or not record[key].strip():
            errors.append(f"{key} must be a non-empty string")
    if not isinstance(record["files"], list) or any(
            not isinstance(item, dict) or set(item) != {"path", "change"}
            or not all(isinstance(item[key], str) and item[key] for key in item)
            for item in record["files"]):
        errors.append("files must contain path/change objects")
    for key in ("expected", "evidence", "tags"):
        if not isinstance(record[key], list) or any(not isinstance(item, str) or not item for item in record[key]):
            errors.append(f"{key} must contain non-empty strings")
    if not record["expected"]:
        errors.append("expected must not be empty")
    if record["prev"] != previous:
        errors.append(f"prev does not match {previous}")
    if record["hash"] != record_hash(record):
        errors.append("hash does not match record content")
    return errors


def removed_git_lines():
    if not (ROOT / ".git").exists():
        return []
    pathspec = ":(glob)docs/history/ledger-*.jsonl"
    removed = []
    for cached in (False, True):
        command = ["git", "diff", "--no-ext-diff", "--unified=0"]
        if cached:
            command.append("--cached")
        command.extend(["--", pathspec])
        result = subprocess.run(command, cwd=ROOT, capture_output=True, text=True, check=False)
        if result.returncode == 0 and any(
                line.startswith("-") and not line.startswith("---")
                for line in result.stdout.splitlines()):
            removed.append("engineering ledger contains removed or replaced lines")
    return removed


def check(directory=HISTORY, enforce_git=True):
    errors, previous, seen = [], GENESIS, set()
    paths = ledger_paths(directory)
    for expected_number, path in enumerate(paths, 1):
        if path.name != f"ledger-{expected_number:04d}.jsonl":
            errors.append(f"{path.name}: ledger segments must be contiguous")
        line_count = len(path.read_text(encoding="utf-8").splitlines())
        if line_count > MAX_SEGMENT_LINES:
            errors.append(f"{path.name}: {line_count} entries exceeds {MAX_SEGMENT_LINES}")
        try:
            entries = [(p, n, item) for p, n, item in read_records(directory) if p == path]
        except ValueError as error:
            errors.append(f"{path.name}: invalid JSON: {error}")
            continue
        for _, number, record in entries:
            for message in validate_record(record, previous, seen):
                errors.append(f"{path.name}:{number}: {message}")
            if isinstance(record, dict):
                seen.add(record.get("id"))
                previous = record.get("hash", previous)
    if enforce_git and directory.resolve() == HISTORY.resolve():
        errors.extend(removed_git_lines())
    return errors


def parse_files(values):
    result = []
    for value in values:
        if "=" not in value:
            raise ValueError("--file must use path=change")
        path, change = value.split("=", 1)
        result.append({"path": path, "change": change})
    return result


def append_record(directory, values, now=None):
    errors = check(directory, enforce_git=False)
    if errors:
        raise ValueError("; ".join(errors))
    existing = list(read_records(directory))
    previous = existing[-1][2]["hash"] if existing else GENESIS
    record = {"id": values.id, "at": (now or datetime.now(timezone.utc)).isoformat(),
              "intent": values.intent, "reason": values.reason,
              "files": parse_files(values.file), "expected": values.expected,
              "evidence": values.evidence, "tags": sorted(set(values.tag)),
              "prev": previous}
    record["hash"] = record_hash(record)
    paths = ledger_paths(directory)
    target = paths[-1] if paths else directory / "ledger-0001.jsonl"
    if target.exists() and len(target.read_text(encoding="utf-8").splitlines()) >= MAX_SEGMENT_LINES:
        target = directory / f"ledger-{len(paths) + 1:04d}.jsonl"
    directory.mkdir(parents=True, exist_ok=True)
    with target.open("a", encoding="utf-8", newline="\n") as stream:
        stream.write(json.dumps(record, ensure_ascii=False, separators=(",", ":")) + "\n")
        stream.flush()
    return record


def query(args):
    found = []
    for _, _, record in read_records(args.directory):
        text = json.dumps(record, ensure_ascii=False)
        if args.id and record["id"] != args.id:
            continue
        if args.tag and args.tag not in record["tags"]:
            continue
        if args.file_path and not any(args.file_path in item["path"] for item in record["files"]):
            continue
        if args.text and args.text.casefold() not in text.casefold():
            continue
        found.append(record)
    return found[-args.limit:]


def build_parser():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    append = commands.add_parser("append")
    append.add_argument("--id", required=True)
    append.add_argument("--intent", required=True)
    append.add_argument("--reason", required=True)
    append.add_argument("--file", action="append", default=[])
    append.add_argument("--expected", action="append", required=True)
    append.add_argument("--evidence", action="append", default=[])
    append.add_argument("--tag", action="append", default=[])
    query_command = commands.add_parser("query")
    query_command.add_argument("--id")
    query_command.add_argument("--tag")
    query_command.add_argument("--file", dest="file_path")
    query_command.add_argument("--text")
    query_command.add_argument("--limit", type=int, default=20)
    commands.add_parser("verify")
    parser.add_argument("--directory", type=Path, default=HISTORY)
    return parser


def main():
    args = build_parser().parse_args()
    if args.command == "append":
        print(json.dumps(append_record(args.directory, args), separators=(",", ":")))
        return 0
    if args.command == "verify":
        errors = check(args.directory)
        print("\n".join(errors) if errors else "Engineering history: PASS")
        return bool(errors)
    if not 1 <= args.limit <= 1000:
        raise SystemExit("--limit must be between 1 and 1000")
    for record in query(args):
        print(json.dumps(record, ensure_ascii=False, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
