"""Record Python's XSD verdicts *with the validated bytes*, for the studio_xsd parity test.

TestHelperMatchesPython (parity_test.go) replays every recorded document through the real
studio_xsd.exe via the Go client and compares status, lines and messages byte for byte.
A dump folder holds entries.json ([{file, schema, label, doctype, status, errors}]) and one
<sha256>.bin per document. Three sources, all run with studio/.venv's Python (lxml 6.0.2):

    python parity_dump.py corpus <out>   # every validate_schema call of golden_gen.py's cases
    python parity_dump.py edge <out>     # malformed/encoding/large inputs (parity_edge.py)
    python parity_dump.py seed <out> <seed.json> <folder>...  # a seed + its source files
    # the whole Python suite, in process (like speed-i2's xsd_record_plugin):
    cd studio/tests; set PYTHONPATH=..\\backend;.;..\\..;<this folder>
    set STUDIO_XSD_DUMP=<out>; python -m pytest -q -p parity_dump

`corpus` also checks that today's verdicts equal the committed scl/testdata/golden/xsd_seed.json.
"""
import hashlib
import json
import os
import sys
import tempfile
from pathlib import Path

HERE = Path(__file__).resolve().parent
SCL_TESTDATA = HERE.parents[2] / "scl" / "testdata"
_SITES = ("studio_app.scl.importer", "studio_app.scl.exporter", "studio_app.api.scl")


class Dump:
    """Collects (bytes, schema) -> verdict, first occurrence wins (the verdict cache's key)."""

    def __init__(self, folder):
        self.folder = Path(folder)
        self.entries = {}

    def add(self, data, schema, verdict, label):
        digest = hashlib.sha256(data).hexdigest()
        if (digest, schema) in self.entries:
            return
        self.folder.mkdir(parents=True, exist_ok=True)
        (self.folder / f"{digest}.bin").write_bytes(data)
        entry = {"file": f"{digest}.bin", "schema": schema, "label": label, "doctype": False}
        entry.update(verdict)
        self.entries[(digest, schema)] = entry

    def write(self):
        listing = [self.entries[key] for key in sorted(self.entries)]
        self.folder.mkdir(parents=True, exist_ok=True)
        (self.folder / "entries.json").write_text(json.dumps(listing, indent=1), encoding="utf-8")
        return listing


def recording(dump, original, label):
    """Wrap validate_schema; keeps only verdicts that reach lxml ("valid"/"invalid")."""
    def validate_schema(source, *args, **kwargs):
        verdict = original(source, *args, **kwargs)
        data = source.encode("utf-8") if isinstance(source, str) else bytes(source)
        if isinstance(verdict, dict) and verdict.get("status") in {"valid", "invalid"}:
            dump.add(data, verdict["schema"], {"status": verdict["status"], "errors": [
                {"line": item["line"], "message": item["message"]} for item in verdict["errors"]]},
                label())
        return verdict
    return validate_schema


def install(dump, label):
    import importlib
    from studio_app.scl import schema_validation
    wrapped = recording(dump, schema_validation.validate_schema, label)
    schema_validation.validate_schema = wrapped
    for name in _SITES:
        module = importlib.import_module(name)
        if hasattr(module, "validate_schema"):
            module.validate_schema = wrapped
    return wrapped


def corpus(out):
    sys.path.insert(0, str(SCL_TESTDATA))
    import pathsetup
    pathsetup.install()
    import golden_gen
    dump, current = Dump(out), {"case": ""}
    validate = install(dump, lambda: "corpus:" + current["case"])
    with tempfile.TemporaryDirectory() as scratch:
        pathsetup.GOLDEN = Path(scratch)  # golden_gen writes its outputs here, not into git
        for case, relative in golden_gen.inputs():
            current["case"] = case
            golden_gen.run_case(case, relative, validate)
    listing = dump.write()
    seed = json.loads((SCL_TESTDATA / "golden" / "xsd_seed.json").read_bytes())
    recorded = {(e["file"][:-4], e["schema"]): (e["status"], e["errors"]) for e in listing}
    committed = {(e["sha256"], e["schema"]): (e["status"], e["errors"]) for e in seed}
    print(f"corpus: {len(listing)} documents; committed seed {len(committed)}; "
          f"same verdicts {sum(recorded.get(k) == v for k, v in committed.items())}; "
          f"missing {len(set(committed) - set(recorded))}; extra {len(set(recorded) - set(committed))}")


def edge(out):
    sys.path.insert(0, str(SCL_TESTDATA))
    import pathsetup
    pathsetup.install()
    from parity_edge import cases, lxml_verdict
    dump = Dump(out)
    for label, schema, data in cases():
        verdict = lxml_verdict(data, schema)
        if verdict.get("doctype"):
            digest = hashlib.sha256(data).hexdigest()
            dump.folder.mkdir(parents=True, exist_ok=True)
            (dump.folder / f"{digest}.bin").write_bytes(data)
            dump.entries[(digest, schema)] = {"file": f"{digest}.bin", "schema": schema,
                                              "label": label, "doctype": True, "status": "", "errors": []}
        else:
            dump.add(data, schema, verdict, label)
    print(f"edge: {len(dump.write())} documents")


def seed(out, seed_file, *document_folders):
    """A Python-recorded seed (xsd_seed.json layout) plus the files it was recorded from."""
    entries = json.loads(Path(seed_file).read_bytes())
    files = {}
    for folder in document_folders:
        for path in sorted(Path(folder).rglob("*")):
            if path.is_file():
                files.setdefault(hashlib.sha256(path.read_bytes()).hexdigest(), path)
    dump, missing = Dump(out), 0
    for entry in entries:
        path = files.get(entry["sha256"])
        if path is None or entry["status"] not in {"valid", "invalid"}:
            missing += 1
            continue
        dump.add(path.read_bytes(), entry["schema"],
                 {"status": entry["status"], "errors": entry["errors"]}, "seed:" + path.name)
    print(f"seed: {len(dump.write())} documents, {missing} seed entries without a document")


# pytest plugin (-p parity_dump): records every verdict of the Python suite with its bytes.
_PLUGIN = {}


def pytest_configure(config):
    out = os.environ.get("STUDIO_XSD_DUMP")
    if out:
        _PLUGIN["dump"] = Dump(out)
        _PLUGIN["test"] = ""
        install(_PLUGIN["dump"], lambda: "pytest:" + _PLUGIN["test"])


def pytest_runtest_setup(item):
    _PLUGIN["test"] = item.nodeid


def pytest_sessionfinish(session, exitstatus):
    if "dump" in _PLUGIN:
        print(f"\nparity_dump: {len(_PLUGIN['dump'].write())} documents")


if __name__ == "__main__":
    {"corpus": corpus, "edge": edge, "seed": seed}[sys.argv[1]](*sys.argv[2:])
