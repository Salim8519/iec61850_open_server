"""Read-only customer-fixture export checks; edited XML stays only in memory."""
import argparse
import hashlib
import json
from pathlib import Path
import random
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "backend"))
from studio_app.scl.document import preview_ieds
from studio_app.scl.exporter import export_ied
from studio_app.scl.importer import import_scl


def audit(folder, limit):
    paths = sorted(path for path in folder.iterdir() if path.suffix.lower() in {".cid", ".icd", ".scd"})
    for path in random.Random(42).sample(paths, min(limit, len(paths))):
        source = path.read_bytes()
        digest = hashlib.sha256(source).hexdigest()
        result = {"file": path.name, "source_sha256": digest}
        try:
            selected = next(item["name"] for item in preview_ieds(source) if not item["unavailable_reason"])
            model = import_scl(source, path.name, [selected])[0]
            point = next(item for item in model["signals"] if item["kind"] == "FLOAT32")
            output = export_ied(source, selected, {point["ref"]: 42.5})
            result.update(ok=True, signal_count=len(model["signals"]),
                          original_ied_count=len(preview_ieds(source)),
                          exported_ied_count=len(preview_ieds(output)),
                          export_sha256=hashlib.sha256(output).hexdigest())
        except Exception as error:
            result.update(ok=False, error=str(error))
        result["source_unchanged"] = hashlib.sha256(path.read_bytes()).hexdigest() == digest
        print(json.dumps(result), flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("folder", type=Path)
    parser.add_argument("--limit", type=int, default=2)
    args = parser.parse_args()
    if args.limit < 1:
        parser.error("Use a positive sample count")
    audit(args.folder, args.limit)
