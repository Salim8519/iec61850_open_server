"""Build the Go SNMP engine, studio_snmp.exe, next to the native workers (ADR-0092).

Usage: python studio/tools/build_snmp_engine.py [--if-stale]
--if-stale rebuilds only when a source of the engine is newer than the executable; the
development launcher uses it so dev never runs an older SNMP engine than the source.
The engine's tests run with the Go engine's tests (engine_tests.py / verify.py).
"""
from pathlib import Path
import argparse
import os
import subprocess
import sys

from build_goose_engine import go_command

STUDIO = Path(__file__).resolve().parents[1]
ENGINE = STUDIO / "engine"
OUTPUT = STUDIO / "native/build/Release/studio_snmp.exe"
SOURCES = (ENGINE / "cmd/studio_snmp", ENGINE / "internal/snmp")


def sources():
    files = [ENGINE / "go.mod"]
    for folder in SOURCES:
        files.extend(path for path in folder.rglob("*") if path.is_file() and not path.name.endswith("_test.go"))
    return files


def stale(output=OUTPUT):
    if not output.exists():
        return True
    built = output.stat().st_mtime
    return any(path.stat().st_mtime > built for path in sources())


def build(output=OUTPUT):
    environment = {**os.environ, "CGO_ENABLED": "0", "GOOS": "windows", "GOARCH": "amd64"}
    output.parent.mkdir(parents=True, exist_ok=True)
    command = [go_command(), "build", "-trimpath", "-ldflags", "-s -w", "-o", str(output), "./cmd/studio_snmp"]
    completed = subprocess.run(command, cwd=ENGINE, env=environment)
    if completed.returncode:
        raise SystemExit(f"go build ./cmd/studio_snmp failed with exit code {completed.returncode}.")
    print(f"Built {output}", flush=True)


def main(arguments):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--if-stale", action="store_true")
    options = parser.parse_args(arguments)
    if options.if_stale and not stale():
        return 0
    build()
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
