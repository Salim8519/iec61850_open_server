"""Build the lab network program, scadabench_labnet.exe, next to the native workers (ADR-0120).

Usage: python studio/tools/build_labnet.py [--if-stale]
The program creates the "ScadaBench Lab" Windows loopback card and its service when the user presses
"Create the lab network card" (one administrator prompt), and, copied to Program Files, runs as that
service, which only adds and removes the simulated IEDs' addresses on the card. It is a windowsgui
program (it never opens a console window).
--if-stale rebuilds only when a source of the program is newer than the executable; the development
launcher uses it so dev never runs an older program than the source.
"""
from pathlib import Path
import argparse
import os
import subprocess
import sys

from build_goose_engine import go_command

STUDIO = Path(__file__).resolve().parents[1]
ENGINE = STUDIO / "engine"
OUTPUT = STUDIO / "native/build/Release/scadabench_labnet.exe"
PACKAGE = "./cmd/scadabench_labnet"
# The program links the labnet package only; a rebuild takes seconds.
SOURCES = (ENGINE / "cmd/scadabench_labnet", ENGINE / "internal/labnet")


def sources():
    files = [ENGINE / "go.mod", ENGINE / "go.sum"]
    for folder in SOURCES:
        files.extend(path for path in folder.rglob("*.go") if not path.name.endswith("_test.go"))
    return [path for path in files if path.is_file()]


def stale(output=OUTPUT):
    if not output.exists():
        return True
    built = output.stat().st_mtime
    return any(path.stat().st_mtime > built for path in sources())


def command(output):
    """The go build command: windowsgui, trimmed, no tags."""
    return [go_command(), "build", "-trimpath", "-ldflags", "-H windowsgui -s -w", "-o", str(output), PACKAGE]


def build(output=OUTPUT):
    environment = {**os.environ, "CGO_ENABLED": "0", "GOOS": "windows", "GOARCH": "amd64"}
    output.parent.mkdir(parents=True, exist_ok=True)
    completed = subprocess.run(command(output), cwd=ENGINE, env=environment)
    if completed.returncode:
        raise SystemExit(f"go build {PACKAGE} failed with exit code {completed.returncode}.")
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
