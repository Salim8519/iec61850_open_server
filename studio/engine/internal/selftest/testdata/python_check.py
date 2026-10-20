"""Python's side of the Go self-test comparison (launcher/self_test.py, frozen engine).

    python -I python_check.py fixture BACKEND OUT
        writes self_test.fixture() (the patched SelfTest template bytes) to OUT.
    python -I python_check.py check BACKEND NATIVE DATA SEED RESULT
        runs self_test.check() on DATA with the workers in NATIVE and writes the printed JSON
        line to RESULT and every XSD verdict it computed (sha256 of the validated bytes) to
        SEED, in the layout of the Go verdict cache seed (studio_xsd.exe is not built in dev).

The recorded testdata (xsd_seed.json, python_check.json) came from the check mode with the
release native workers of change 0139 and studio/.venv's Python.
"""
import hashlib
import importlib
import json
import sys
from pathlib import Path


def fixture(backend, out):
    sys.path.insert(0, backend)
    from studio_app.launcher.self_test import fixture as make
    Path(out).write_bytes(make())


def check(backend, native, data, seed_out, result_out):
    sys.path.insert(0, backend)
    from studio_app.scl import schema_validation
    seen = {}
    original = schema_validation.validate_schema

    def recording(source, *args, **kwargs):
        verdict = original(source, *args, **kwargs)
        payload = source.encode("utf-8") if isinstance(source, str) else bytes(source)
        if isinstance(verdict, dict) and verdict.get("status") in {"valid", "invalid"}:
            seen.setdefault(hashlib.sha256(payload).hexdigest(), {
                "schema": verdict["schema"], "status": verdict["status"], "errors": verdict["errors"]})
        return verdict

    schema_validation.validate_schema = recording
    for name in ("studio_app.scl.importer", "studio_app.scl.exporter", "studio_app.api.scl"):
        importlib.import_module(name).validate_schema = recording
    from studio_app.settings import Settings
    from studio_app.launcher.self_test import check as run
    settings = Settings(Path(data), Path(native) / "studio_ied.exe", Path(backend).parent / "frontend" / "dist")
    text = json.dumps(run(settings))
    Path(result_out).write_text(text, encoding="utf-8")
    seed = [{"sha256": digest, **verdict} for digest, verdict in sorted(seen.items())]
    Path(seed_out).write_text(json.dumps(seed), encoding="utf-8")


if __name__ == "__main__":
    if sys.argv[1] == "fixture":
        fixture(*sys.argv[2:4])
    else:
        check(*sys.argv[2:7])
