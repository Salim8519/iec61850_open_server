"""Validate a saved project before restart; optionally preserve its exact bytes."""
import argparse
import hashlib
import json
from pathlib import Path
import sys
import tempfile
from uuid import uuid4

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "studio/backend"))
from studio_app.projects.archive import MAX_UNPACKED, decode
from studio_app.projects.document import compact
from studio_app.projects.service import ProjectService


def comparable(project):
    result = compact(project)
    result.setdefault("test_panel", [])
    return result


def check(source, backup=False):
    source = source.resolve(strict=True)
    if source.stat().st_size > MAX_UNPACKED:
        raise ValueError("Archive exceeds the project size limit")
    payload = source.read_bytes()
    digest = hashlib.sha256(payload).hexdigest()
    with tempfile.TemporaryDirectory(prefix="project-check-", dir=ROOT / ".local") as temporary:
        projects = ProjectService(Path(temporary))
        projects.restore(payload, persist=False)
        compact_payload = projects.export()
        engineering = compact(projects.project)
        projects.restore(compact_payload, persist=False)
        if compact(projects.project) != engineering:
            raise RuntimeError("Compact archive changed engineering state during reopen")
        if comparable(decode(payload)) != comparable(decode(compact_payload)):
            raise RuntimeError("Restoration changed stored engineering state")
        snapshot = projects.snapshot()
        result = {"valid": True, "sha256": digest, "project_id": snapshot["id"],
                  "original_bytes": len(payload), "compact_bytes": len(compact_payload),
                  "compact_roundtrip_verified": True,
                  "ieds": len(snapshot["ieds"]), "equipment": len(snapshot["diagram"]["nodes"]),
                  "sources": len(snapshot["sources"]), "scenarios": len(snapshot["scenarios"])}
    if backup:
        if hashlib.sha256(source.read_bytes()).hexdigest() != digest:
            raise RuntimeError("Project changed during validation; save and validate again before restart")
        directory = source.parent / "recovery"
        if directory.is_symlink():
            raise RuntimeError("Recovery folder must not be a symbolic link")
        directory.mkdir(exist_ok=True)
        target = directory / ("before-backend-restart-" + uuid4().hex + ".subsim")
        with target.open("xb") as stream:
            stream.write(payload)
        if hashlib.sha256(target.read_bytes()).hexdigest() != digest:
            raise RuntimeError("Restart backup failed verification")
        result["backup"] = str(target)
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("project", type=Path)
    parser.add_argument("--backup", action="store_true")
    args = parser.parse_args()
    print(json.dumps(check(args.project, args.backup)), flush=True)
