"""Run the Go engine module's tests (studio/engine, ADR-0074); used by verify.py.

The pyjson differential test compares against CPython and Flask, so it runs with this
Python (STUDIO_PYTHON). Usage: python studio/tools/engine_tests.py [--cases N]
"""
from pathlib import Path
import argparse
import os
import subprocess
import sys

from build_goose_engine import go_command

STUDIO = Path(__file__).resolve().parents[1]
ENGINE = STUDIO / "engine"
VENV_PYTHON = STUDIO / ".venv/Scripts/python.exe"  # has Flask; a bare system Python may not


def test(cases=None):
    python = VENV_PYTHON if VENV_PYTHON.exists() else Path(sys.executable)
    environment = {**os.environ, "CGO_ENABLED": "0", "STUDIO_PYTHON": str(python)}
    if cases:
        environment["PYJSON_CASES"] = str(cases)
    # -trimpath: the antivirus here refuses some untrimmed test binaries ("Access is denied");
    # -p 2 keeps the laptop usable while the suite runs.
    command = [go_command(), "test", "-trimpath", "-p", "2", "./...", "-timeout", "60m"]
    completed = subprocess.run(command, cwd=ENGINE,
                               env=environment, capture_output=True, text=True)
    if completed.returncode:
        return [f"studio/engine: go test failed\n{completed.stdout}{completed.stderr}".strip()]
    return []


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--cases", type=int, help="pyjson differential cases (P0 gate: 1000000)")
    errors = test(parser.parse_args().cases)
    print("\n".join(errors) or "studio/engine: PASS")
    raise SystemExit(bool(errors))
