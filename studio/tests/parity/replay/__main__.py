"""Golden replay commands (run from studio/tests with PYTHONPATH=../backend;.):

python -m parity.replay record   --engine python-http [scenario ...]   # write golden/transcripts
python -m parity.replay golden   --engine go [scenario ...]            # replay vs committed goldens
python -m parity.replay dual     --engines python-http,go [scenario ...]
python -m parity.replay selfdiff [scenario ...]    # python-http with PYTHONHASHSEED 0 and 1
Exit code 0 only with zero differences (and zero NaN occurrences).
"""
import argparse
import json
from pathlib import Path
import sys
import tempfile

from parity.replay.diff import diff, nan_count, summary
from parity.replay.record import Recorder

GOLDEN = Path(__file__).resolve().parents[1] / "golden"
SCENARIOS, TRANSCRIPTS = GOLDEN / "scenarios", GOLDEN / "transcripts"


def scenarios(names):
    paths = sorted(SCENARIOS.rglob("*.json")) if not names else [Path(name) if Path(name).is_file()
                                                                  else SCENARIOS / f"{name}.json" for name in names]
    return [(path.relative_to(SCENARIOS).with_suffix("").as_posix(), json.loads(path.read_text(encoding="utf-8")))
            for path in paths]


def read(path):
    return [json.loads(line) for line in Path(path).read_text(encoding="utf-8").splitlines() if line]


def record(engine, key, scenario, folder, env=None):
    output = Path(folder) / f"{key}.jsonl"
    Recorder(engine, env=env).run(scenario, output)
    return read(output)


def report(key, left, right, names):
    lines = diff(left, right, names, key)
    for _, line in lines:
        print(line)
    nans = nan_count(left) + nan_count(right)
    if nans:
        print(f"{key}: {nans} NaN/Infinity occurrence(s): Python bug to fix first")
    print(f"{key}: {len(lines)} difference(s) {summary(lines)}")
    return len(lines) + nans


def main(argv=None):
    parser = argparse.ArgumentParser(description="ScadaBench golden replay")
    parser.add_argument("command", choices=["record", "golden", "dual", "selfdiff"])
    parser.add_argument("names", nargs="*")
    parser.add_argument("--engine", default="python-http")
    parser.add_argument("--engines", default="python-http,python-http")
    args = parser.parse_args(argv)
    failures = 0
    for key, scenario in scenarios(args.names):
        if args.command == "record":
            transcript = record(args.engine, key, scenario, TRANSCRIPTS)
            failures += nan_count(transcript)
            print(f"recorded {key}: {len(transcript) - 1} steps")
            continue
        with tempfile.TemporaryDirectory(prefix="studio-replay-") as scratch:
            if args.command == "golden":
                left, names = read(TRANSCRIPTS / f"{key}.jsonl"), ("golden", args.engine)
                right = record(args.engine, key, scenario, Path(scratch) / "b")
            elif args.command == "dual":
                names = tuple(args.engines.split(","))
                left = record(names[0], key, scenario, Path(scratch) / "a")
                right = record(names[1], key, scenario, Path(scratch) / "b")
            else:
                names = ("seed0", "seed1")
                left = record("python-http", key, scenario, Path(scratch) / "a", {"PYTHONHASHSEED": "0"})
                right = record("python-http", key, scenario, Path(scratch) / "b", {"PYTHONHASHSEED": "1"})
            failures += report(key, left, right, names)
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
