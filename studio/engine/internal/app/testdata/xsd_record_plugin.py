"""pytest plugin: record every XSD verdict the frozen Python app computes (dev only).

studio_xsd.exe is not built yet (gap 1), so the Go engine's suite run (STUDIO_ENGINE=go)
needs Python's verdicts for every SCL text the tests import. Run the Python suite once in
process with this plugin, then hand the file to the Go run as STUDIO_XSD_SEED:

    cd studio/tests
    set PYTHONPATH=..\\backend;.;..\\..;..\\engine\\internal\\app\\testdata
    set STUDIO_XSD_RECORD=recorded.json
    python -m pytest -q -p xsd_record_plugin

The seed has the layout of xsd_verdicts.py (sha256 of the validated bytes, schema, status,
errors); only "valid"/"invalid" verdicts are kept, like the Go verdict cache.
"""
import hashlib
import json
import os
from pathlib import Path

_SITES = ("studio_app.scl.importer", "studio_app.scl.exporter", "studio_app.api.scl")
_SEEN = {}


def _recording(original):
    def validate_schema(source, *args, **kwargs):
        verdict = original(source, *args, **kwargs)
        data = source.encode("utf-8") if isinstance(source, str) else bytes(source)
        if isinstance(verdict, dict) and verdict.get("status") in {"valid", "invalid"}:
            _SEEN.setdefault(hashlib.sha256(data).hexdigest(), {
                "schema": verdict["schema"], "status": verdict["status"], "errors": verdict["errors"]})
        return verdict
    return validate_schema


def pytest_configure(config):
    import importlib
    from studio_app.scl import schema_validation
    wrapped = _recording(schema_validation.validate_schema)
    schema_validation.validate_schema = wrapped
    for name in _SITES:
        importlib.import_module(name).validate_schema = wrapped


def pytest_sessionfinish(session, exitstatus):
    out = os.environ.get("STUDIO_XSD_RECORD")
    if out:
        seed = [{"sha256": digest, **verdict} for digest, verdict in sorted(_SEEN.items())]
        Path(out).write_text(json.dumps(seed), encoding="utf-8")
