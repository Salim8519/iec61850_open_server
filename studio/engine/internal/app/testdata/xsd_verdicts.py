"""Write Python's (lxml 6.0.2) XSD verdicts as a seed for the Go engine's verdict cache.

Usage: python -I xsd_verdicts.py --backend <studio/backend> --out SEED.json INPUT...
An INPUT ending in .json is a project tree (every ieds[].source text is validated);
"template:NAME" is make_template(NAME); any other INPUT is an SCL file validated as its bytes. studio_xsd.exe is not built yet, so the
composed engine's tests and the dev run import these verdicts (STUDIO_XSD_SEED).
"""
import argparse
import hashlib
import json
import sys
from pathlib import Path


def sources(path):
    if str(path).startswith("template:"):
        from studio_app.scl.templates import make_template
        source = make_template(str(path)[len("template:"):])
        return [source.encode("utf-8") if isinstance(source, str) else source]
    if path.suffix == ".json":
        project = json.loads(path.read_text(encoding="utf-8"))
        return [ied["source"].encode("utf-8") for ied in project["ieds"] if ied.get("source")]
    return [path.read_bytes()]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--backend", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("inputs", nargs="+")
    args = parser.parse_args()
    sys.path.insert(0, args.backend)
    from studio_app.scl.schema_validation import validate_schema
    seed, seen = [], set()
    for name in args.inputs:
        for data in sources(Path(name)):
            digest = hashlib.sha256(data).hexdigest()
            if digest in seen:
                continue
            seen.add(digest)
            verdict = validate_schema(data)
            if verdict["status"] in {"valid", "invalid"}:
                seed.append({"sha256": digest, "schema": verdict["schema"], "status": verdict["status"],
                             "errors": verdict["errors"]})
    Path(args.out).write_text(json.dumps(seed), encoding="utf-8")
    print(f"{len(seed)} verdicts", flush=True)


if __name__ == "__main__":
    main()
