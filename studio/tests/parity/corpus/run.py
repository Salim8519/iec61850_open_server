"""Run corpus cases through engines over HTTP and diff every output (C7).

python -m parity.corpus.run selfdiff [--cases a,b] [--limit N]   # python-http, hash seeds 0 and 1
python -m parity.corpus.run dual --engines python-http,go [--cases ...]
python -m parity.corpus.run record [--cases ...] --out <folder>   # Python goldens for stream B
Each case: preview, validate, import (every IED name in the file), project, signals, source,
export-scl, project download - one fresh engine per case. Exit 0 only with zero differences.
"""
import argparse
import json
from pathlib import Path
import sys
import tempfile

from parity.corpus.scl_corpus import MANIFEST
from parity.replay.diff import diff, nan_count, summary
from parity.replay.manifest import Manifest
from parity.replay.record import Recorder


def scenario(case, names):
    upload = {"file": f"@corpus:{case}"}
    return {"name": f"corpus-{case}", "steps": [
        {"req": "POST /api/v1/session", "json": {"token": "golden-session-token"}},
        {"req": "POST /api/v1/scl/preview", "multipart": upload},
        {"req": "POST /api/v1/scl/validate", "multipart": upload},
        {"req": "POST /api/v1/scl/import", "multipart": upload, "form": {"selected_names": json.dumps(names)}},
        {"req": "GET /api/v1/project", "bind": {"ied": "ieds.0.id"}},
        {"req": "GET /api/v1/ieds/{$ied}/signals", "query": {"limit": 500}, "needs": "ied"},
        {"req": "GET /api/v1/ieds/{$ied}/source", "needs": "ied"},
        {"req": "POST /api/v1/ieds/{$ied}/export-scl", "json": {"overrides": {}}, "needs": "ied"},
        {"req": "GET /api/v1/project/download"}]}


def selected(args):
    cases = [entry["case"] for entry in json.loads(MANIFEST.read_text(encoding="utf-8"))["cases"]]
    if args.cases:
        wanted = args.cases.split(",")
        cases = [case for case in cases if case in wanted]
    return cases[:args.limit] if args.limit else cases


def run_case(engine, case, manifest, folder, env=None):
    names = manifest.inputs[f"corpus:{case}"]["ieds"]
    output = Path(folder) / f"{case}.jsonl"
    Recorder(engine, manifest, env=env).run(scenario(case, names), output)
    return [json.loads(line) for line in output.read_text(encoding="utf-8").splitlines()]


def main(argv=None):
    parser = argparse.ArgumentParser(description="ScadaBench SCL corpus runner")
    parser.add_argument("command", choices=["selfdiff", "dual", "record"])
    parser.add_argument("--engines", default="python-http,python-http")
    parser.add_argument("--cases")
    parser.add_argument("--limit", type=int)
    parser.add_argument("--out", type=Path)
    args = parser.parse_args(argv)
    manifest, total = Manifest(), 0
    for case in selected(args):
        if args.command == "record":
            run_case("python-http", case, manifest, args.out)
            print(f"recorded {case}")
            continue
        with tempfile.TemporaryDirectory(prefix="studio-corpus-") as scratch:
            if args.command == "selfdiff":
                names = ("seed0", "seed1")
                left = run_case("python-http", case, manifest, Path(scratch) / "a", {"PYTHONHASHSEED": "0"})
                right = run_case("python-http", case, manifest, Path(scratch) / "b", {"PYTHONHASHSEED": "1"})
            else:
                names = tuple(args.engines.split(","))
                left = run_case(names[0], case, manifest, Path(scratch) / "a")
                right = run_case(names[1], case, manifest, Path(scratch) / "b")
        lines, nans = diff(left, right, names, case), nan_count(left) + nan_count(right)
        for _, line in lines:
            print(line)
        print(f"{case}: {len(lines)} difference(s) {summary(lines)}" + (f", {nans} NaN" if nans else ""), flush=True)
        total += len(lines)
    return 1 if total else 0


if __name__ == "__main__":
    sys.exit(main())
