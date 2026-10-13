"""Read-only compatibility sampling; outputs metadata, never raw customer SCL."""
import argparse
import hashlib
import json
from pathlib import Path
import random
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "backend"))
from studio_app.scl.importer import import_scl


def audit(folder, limit):
    paths = [path for path in folder.iterdir() if path.suffix.lower() in {".cid", ".icd", ".scd"}]
    paths = random.Random(42).sample(sorted(paths), min(limit, len(paths)))
    results = []
    for path in paths:
        source = path.read_bytes()
        result = {"file": path.name, "bytes": len(source), "sha256": hashlib.sha256(source).hexdigest()}
        try:
            models = import_scl(source, path.name)
            result.update(ok=True, ieds=len(models), signals=sum(len(m["signals"]) for m in models),
                          controls=sum(len(m["controls"]) for m in models),
                          xsd_status=models[0]["validation"]["status"],
                          xsd_errors=len(models[0]["validation"]["errors"]))
        except Exception as error:
            result.update(ok=False, error=str(error))
        results.append(result)
    return results


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("folder", type=Path)
    parser.add_argument("--limit", type=int, default=6)
    args = parser.parse_args()
    print(json.dumps(audit(args.folder, args.limit), indent=2))
