"""Build the virtual machine helper, scadabench_hostkey.exe, next to the native workers (ADR-0099).

Usage: python studio/tools/build_hostkey_helper.py [--if-stale]
The helper lets the virtual machines of a licensed PC use its lifetime license; it only runs when
the user switches "Let my virtual machines use this license" on in Settings. It is a windowsgui
program (it starts at Windows logon and must never open a console window). It is always built
WITHOUT build tags: the test seam that binds a loopback address (tag hostkeytest) never reaches a
program a user can run.
--if-stale rebuilds only when a source of the helper is newer than the executable; the development
launcher uses it so dev never runs an older helper than the source.
"""
from pathlib import Path
import argparse
import os
import subprocess
import sys

from build_goose_engine import go_command

STUDIO = Path(__file__).resolve().parents[1]
ENGINE = STUDIO / "engine"
OUTPUT = STUDIO / "native/build/Release/scadabench_hostkey.exe"
PACKAGE = "./cmd/scadabench_hostkey"
# The helper links the license package and what it uses, so any Go source of the engine module
# (other than tests) can change the program; a rebuild takes seconds.
SOURCES = (ENGINE / "cmd/scadabench_hostkey", ENGINE / "internal")


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
    """The go build command: windowsgui, trimmed, no tags (never the hostkeytest seam)."""
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
