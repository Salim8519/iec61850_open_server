"""Shared helpers for the Python golden generators of Go stream D (live acquisition).

Run with the studio venv from the repository root, for example:
    studio\\.venv\\Scripts\\python.exe studio\\engine\\internal\\acquisition\\testdata\\gen\\all.py
Only public SCL files from scd/ are used (open_substation, model_substation).
"""
import copy
import io
import json
import shutil
import sys
import tempfile
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[5]
OUT = HERE.parent / "generated"
sys.path.insert(0, str(ROOT / "studio" / "backend"))

from flask import Flask, jsonify  # noqa: E402

_FLASK = Flask("golden")
_PROJECTS = {}

PUBLIC = {
    "open": ("open_substation.scd", ["IED1_XCBR", "IED2_PTOC"]),
    "model": ("model_substation.scd", ["FEED1", "BUS1"]),
}


def flask(value):
    """The exact bytes Flask's jsonify writes for value."""
    with _FLASK.app_context():
        return jsonify(value).get_data(as_text=True)


def app_with(name):
    """A Flask Studio app (temporary data folder) with one public SCD imported."""
    from studio_app.app_factory import create_app
    from studio_app.settings import Settings
    filename, selected = PUBLIC[name]
    folder = Path(tempfile.mkdtemp(prefix="golden-"))
    app = create_app(Settings(folder, folder / "unused.exe", folder), "golden")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "golden"})
    source = (ROOT / "scd" / filename).read_bytes()
    answer = client.post("/api/v1/scl/import", data={"file": (io.BytesIO(source), "station.scd"),
                                                     "selected_names": json.dumps(selected)})
    assert answer.status_code == 200, answer.get_data(as_text=True)
    return app, client, folder


def close(app, folder):
    app.extensions["runtime"].close()
    shutil.rmtree(folder, ignore_errors=True)


def project(name, keep_source=False):
    """A deep copy of the imported public project; source/cfg dropped unless asked for."""
    if name not in _PROJECTS:
        app, _, folder = app_with(name)
        _PROJECTS[name] = copy.deepcopy(app.extensions["projects"].project)
        close(app, folder)
    result = copy.deepcopy(_PROJECTS[name])
    if not keep_source:
        for ied in result["ieds"]:
            ied.pop("source", None)
            ied.pop("cfg", None)
    return result


def plain(value):
    """JSON-able copy: tuples become lists, sets sorted lists, tuple keys joined lists."""
    return json.loads(json.dumps(value, default=lambda item: sorted(item) if isinstance(item, set) else str(item)))


def record(file, cases):
    """cases: [(name, args list, want text)]."""
    OUT.mkdir(parents=True, exist_ok=True)
    items = [{"name": name, "args": plain(args), "want": want} for name, args, want in cases]
    (OUT / file).write_text(json.dumps(items, indent=None), encoding="utf-8", newline="\n")
    print(f"{file}: {len(items)} cases")
