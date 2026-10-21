"""Test-wide setup: the app's daily update check must not reach the internet during tests.

Engine switch (Go engine plan, docs/architecture/go-engine/tests/03-parity-harness.md):
STUDIO_ENGINE=python (default) runs every test exactly as before, in process. python-http
and go start a real backend process per create_app call through tests/parity, so the same
test files judge both engines; tests marked `inprocess` are skipped there and counted.
"""
import os
from pathlib import Path
import sys

os.environ["SCADABENCH_UPDATE_CHECK"] = "off"
ENGINE = os.environ.get("STUDIO_ENGINE", "python")
if ENGINE not in ("python", "python-http", "go"):
    raise SystemExit(f"STUDIO_ENGINE must be python, python-http or go, not {ENGINE!r}")


def pytest_configure(config):
    config.addinivalue_line("markers", "inprocess: needs Python objects; skipped for HTTP engines")
    config.addinivalue_line("markers", "engine_neutral: native/wire contract; the engine only supplies cfg")
    if ENGINE != "python":
        sys.path.insert(0, str(Path(__file__).resolve().parent))
        import parity.plugin  # replaces create_app before any test module imports it
        config.pluginmanager.register(parity.plugin, "studio-parity")
