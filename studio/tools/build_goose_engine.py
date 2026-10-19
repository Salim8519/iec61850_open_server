"""Test and build the Go GOOSE engine, studio_goose.exe, next to the native workers (ADR-0073).

Usage: python studio/tools/build_goose_engine.py [--if-stale] [--skip-tests]
--if-stale rebuilds only when a Go source is newer than the executable; the
development launcher uses it so dev never runs an older engine than the source.
"""
from pathlib import Path
import argparse
import os
import shutil
import subprocess
import sys

STUDIO = Path(__file__).resolve().parents[1]
ENGINE = STUDIO / "goose_engine"
OUTPUT = STUDIO / "native/build/Release/studio_goose.exe"


def go_command():
    found = shutil.which("go")
    if found:
        return found
    default = Path(os.environ.get("ProgramFiles", r"C:\Program Files")) / "Go/bin/go.exe"
    if default.exists():
        return str(default)
    raise SystemExit("Install Go 1.24 or newer (https://go.dev/dl/) to build studio_goose.exe.")


def sources():
    return [path for path in ENGINE.rglob("*") if path.suffix in {".go", ".mod", ".sum"}]


def stale(output=OUTPUT):
    if not output.exists():
        return True
    built = output.stat().st_mtime
    return any(path.stat().st_mtime > built for path in sources())


def run(arguments, environment):
    completed = subprocess.run(arguments, cwd=ENGINE, env=environment)
    if completed.returncode:
        raise SystemExit(f"{' '.join(arguments[1:3])} failed with exit code {completed.returncode}.")


def test():
    """Run the engine's Go tests; used by verify.py."""
    environment = {**os.environ, "CGO_ENABLED": "0"}
    completed = subprocess.run([go_command(), "test", "./..."], cwd=ENGINE, env=environment,
                               capture_output=True, text=True)
    if completed.returncode:
        return [f"studio/goose_engine: go test failed\n{completed.stdout}{completed.stderr}".strip()]
    return []


def build(output=OUTPUT, run_tests=True):
    environment = {**os.environ, "CGO_ENABLED": "0", "GOOS": "windows", "GOARCH": "amd64"}
    go = go_command()
    if run_tests:
        run([go, "test", "./..."], environment)
    output.parent.mkdir(parents=True, exist_ok=True)
    run([go, "build", "-trimpath", "-ldflags", "-s -w", "-o", str(output), "./cmd/studio_goose"], environment)
    print(f"Built {output}", flush=True)


def main(arguments):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--if-stale", action="store_true")
    parser.add_argument("--skip-tests", action="store_true")
    options = parser.parse_args(arguments)
    if options.if_stale and not stale():
        return 0
    build(run_tests=not options.skip_tests)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
