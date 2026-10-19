"""Write Python goldens for every public fixture and corpus case into golden/<case>/.

Run after corpus_gen.py: python studio/engine/internal/scl/testdata/golden_gen.py
Every output comes from calling the frozen Python functions directly (see README.md).
"""
import hashlib
import json
import shutil
import pathsetup

pathsetup.install()

from golden_common import XsdRecorder, capture, dumps, normalise, write_json  # noqa: E402
from golden_export import export_errors, export_one  # noqa: E402
from studio_app.scl.document import preview_ieds  # noqa: E402
from studio_app.scl.importer import import_scl  # noqa: E402

FIXTURES = [("fixture_model_substation", "scd/model_substation.scd"),
            ("fixture_open_substation", "scd/open_substation.scd"),
            ("fixture_protection_relay", "scd/protection_relay.scd"),
            ("fixture_demo", "marketing/videos/demo/scadabench_demo.scd")]
CORPUS_PREFIX = "studio/engine/internal/scl/testdata/corpus/"
SELECT_PROBES = ("empty", "unknown", "duplicate", "non_string")


def inputs():
    result = [(case, path) for case, path in FIXTURES]
    manifest = json.loads((pathsetup.CORPUS / "manifest.json").read_bytes())
    result.extend((entry["case"], CORPUS_PREFIX + entry["case"] + ".scd") for entry in manifest)
    return result


def ied_names(data):
    """Names in document order, from the parsed document (may include None)."""
    from studio_app.scl.document import parse_document
    ok, root = capture(parse_document, data)
    return [ied.get("name") for ied in root.findall("IED")][:3] if ok else []


def write_result(folder, name, result, cfgs, files, cfg_prefix):
    """The (normalised) import result or error dict, plus each cfg as exact bytes."""
    for index, cfg in enumerate(cfgs):
        pathsetup.write_bytes(folder / f"{cfg_prefix}{index}.cfg", cfg.encode("utf-8"))
        files.append(f"{cfg_prefix}{index}.cfg")
    write_json(folder, name, result, files)


def select_errors(data, names):
    first = names[0] if names and isinstance(names[0], str) else "X"
    probes = [[], ["nope"], [first, first], [1]]
    results = []
    for probe in probes:
        ok, result = capture(import_scl, data, "case.scd", probe)
        if ok:
            normalise(result)
        results.append({"selected_names": probe, "result" if ok else "error": result})
    return results


def run_case(case, relative, validate):
    folder = pathsetup.GOLDEN / case
    data = (pathsetup.REPO / relative).read_bytes()
    files, aliases = [], {}
    pathsetup.write_bytes(folder / "input.sha256", hashlib.sha256(data).hexdigest().encode("ascii"))
    files.append("input.sha256")
    ok, preview = capture(preview_ieds, data)
    write_json(folder, "preview.json" if ok else "error.json", preview, files)
    write_json(folder, "validate.json", capture(validate, data)[1], files)
    ok, result = capture(import_scl, data, "case.scd")
    write_result(folder, "import-all.json", result, normalise(result) if ok else [], files, "cfg-")
    names = ied_names(data)
    text_ok, text = capture(lambda raw: raw.decode("utf-8-sig"), data)
    exported = []
    for index, name in enumerate(names):
        ok, result = capture(import_scl, data, "case.scd", [name])
        cfgs = normalise(result) if ok else []
        if (folder / "import-all.json").read_bytes() == dumps(result):
            aliases[f"import-each-{index}.json"] = "import-all.json"
        else:
            write_result(folder, f"import-each-{index}.json", result, cfgs, files, f"cfg-each-{index}-")
        if ok and text_ok and result[0]["signals"]:
            if export_one(folder, case, index, text, name, result[0]["signals"], files):
                exported.append((name, result[0]["signals"]))
    write_json(folder, "import-select-errors.json", select_errors(data, names), files)
    if exported:
        export_errors(folder, text, exported[0][0], exported[0][1], files)
    return {"case": case, "input": relative, "sha256": hashlib.sha256(data).hexdigest(),
            "ied_names": names, "files": sorted(files), "aliases": aliases}


def clean():
    if pathsetup.GOLDEN.exists():
        for child in pathsetup.GOLDEN.iterdir():
            if child.is_dir():
                shutil.rmtree(child)
    for name in ("manifest.json", "xsd_seed.json"):
        (pathsetup.GOLDEN / name).unlink(missing_ok=True)


def main():
    recorder = XsdRecorder()
    validate = recorder.install()
    clean()
    manifest = [run_case(case, relative, validate) for case, relative in inputs()]
    pathsetup.write_bytes(pathsetup.GOLDEN / "manifest.json", dumps(manifest))
    pathsetup.write_bytes(pathsetup.GOLDEN / "xsd_seed.json", dumps(recorder.seed()))
    total = sum(len(entry["files"]) for entry in manifest)
    print(f"golden: {len(manifest)} cases, {total} files, {len(recorder.seed())} XSD seed entries")


if __name__ == "__main__":
    main()
