"""Locate the worktree and put the frozen Python backend on sys.path (import side effect free)."""
import sys
from pathlib import Path

TESTDATA = Path(__file__).resolve().parent
REPO = TESTDATA.parents[4]
BACKEND = REPO / "studio" / "backend"
UNIT_TESTS = REPO / "studio" / "tests" / "unit"
CORPUS = TESTDATA / "corpus"
GOLDEN = TESTDATA / "golden"


def install():
    """Make studio_app and the public unit-test fixture helpers importable."""
    for path in (BACKEND, TESTDATA):
        if str(path) not in sys.path:
            sys.path.insert(0, str(path))
    if str(UNIT_TESTS) not in sys.path:
        sys.path.append(str(UNIT_TESTS))


def write_bytes(path, data):
    """Write raw bytes exactly (no newline translation)."""
    path.parent.mkdir(parents=True, exist_ok=True)
    with open(path, "wb") as handle:
        handle.write(data)

