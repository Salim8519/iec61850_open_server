"""Run the frozen Python startup rename of old .subsim files on a data folder (python_test.go).

  python migrate.py <data_dir>   prints {"records": [...], "journal": [...], "files": {path: sha256}}

Run with studio/.venv/Scripts/python.exe; studio/backend is put on sys.path from this script's
location. The Go test builds the same folder twice, runs Go on one and this script on the other
and compares what is left on disk and what was journaled.
"""
import hashlib
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[5] / "backend"))

from studio_app.projects.extension_migration import migrate  # noqa: E402


def main(folder):
    try:
        records = migrate(folder)
        stopped = None
    except RuntimeError as error:
        records, stopped = [], str(error)
    journal = folder / "logs" / "project-file-migrations.jsonl"
    lines = [json.loads(line) for line in journal.read_text(encoding="utf-8").splitlines()] if journal.exists() else []
    files = {path.relative_to(folder).as_posix(): hashlib.sha256(path.read_bytes()).hexdigest()
             for path in sorted(folder.rglob("*")) if path.is_file() and path != journal}
    print(json.dumps({"records": records, "journal": lines, "files": files, "stopped": stopped}))


if __name__ == "__main__":
    main(Path(sys.argv[1]))
