"""Write the SCL corpus: fixtures/<case>.scd plus fixtures/manifest.json (deterministic).

Run from studio/tests with the backend on PYTHONPATH:
    python -m parity.corpus.scl_corpus --write     # regenerate inputs + manifest
    python -m parity.corpus.scl_corpus --check     # committed files still match the generator
`expect` is what the frozen Python importer does with the case (accept, warn or reject),
recorded at generation so a Go engine is judged against it. Cases marked `generated`
(the 20 MB size cases) are not committed; their bytes are rebuilt on demand and hash-checked.
"""
import argparse
import hashlib
import json
from pathlib import Path
import sys

from parity.corpus import cases_bytes, cases_comms, cases_model, cases_values, cases_xml

CASES = Path(__file__).resolve().parent / "fixtures"
MANIFEST = CASES / "manifest.json"


def all_cases():
    found = []
    for module in (cases_bytes, cases_xml, cases_model, cases_values, cases_comms):
        found.extend(module.cases())
    names = [case.case.lower() for case in found]  # Windows file names ignore case
    duplicates = sorted({name for name in names if names.count(name) > 1})
    if duplicates:
        raise ValueError(f"duplicate corpus case names: {duplicates}")
    return found


def expectation(data):
    from studio_app.scl.importer import import_scl
    try:
        models = import_scl(data, "corpus.scd")
    except Exception as error:  # every refusal reaches the user as a 400
        return "reject", type(error).__name__
    warned = any(model.get("warnings") for model in models)
    return ("warn" if warned else "accept"), None


def document(cases):
    entries = []
    for case in cases:
        expect, error = expectation(case.data)
        entries.append({"case": case.case, "purpose": case.purpose, "base": case.base,
                        "file": f"{case.case}.scd", "sha256": hashlib.sha256(case.data).hexdigest(),
                        "size": len(case.data), "generated": case.generated, "expect": expect, "error": error})
    return {"version": 1, "cases": entries}


def bytes_for(name):
    """Input bytes of one case: the committed file, or rebuilt for generated cases."""
    entry = next(item for item in json.loads(MANIFEST.read_text(encoding="utf-8"))["cases"] if item["case"] == name)
    if entry["generated"]:
        data = next(case.data for case in all_cases() if case.case == name)
    else:
        data = (CASES / entry["file"]).read_bytes()
    if hashlib.sha256(data).hexdigest() != entry["sha256"]:
        raise ValueError(f"corpus case {name} does not match its manifest hash")
    return data


def main(argv=None):
    parser = argparse.ArgumentParser(description="ScadaBench SCL corpus generator")
    parser.add_argument("--write", action="store_true")
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args(argv)
    cases = all_cases()
    built = document(cases)
    if args.write:
        CASES.mkdir(parents=True, exist_ok=True)
        for case in cases:
            if not case.generated:
                (CASES / f"{case.case}.scd").write_bytes(case.data)
        MANIFEST.write_text(json.dumps(built, indent=2) + "\n", encoding="utf-8")
    if args.check:
        stored = json.loads(MANIFEST.read_text(encoding="utf-8"))
        if stored != built:
            print("corpus manifest differs from the generator; run --write and review the diff")
            return 1
    totals = {}
    for entry in built["cases"]:
        totals[entry["expect"]] = totals.get(entry["expect"], 0) + 1
    print(f"{len(cases)} cases {totals}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
