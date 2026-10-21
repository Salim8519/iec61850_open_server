"""Rename the app's own project files from .subsim to .scadabench once, at startup (ADR-0096).

Only files ScadaBench itself wrote are touched: <data>/autosave.subsim and <data>/recovery/*.subsim.
Nothing is deleted or overwritten. When the new name already exists both files are kept and the
fact is journaled. Every outcome is appended to <data>/logs/project-file-migrations.jsonl, an
append-only journal of one JSON line per file (the Go engine writes the same lines).
"""
import json
import logging
import os
from datetime import datetime, timezone
from pathlib import Path
from uuid import uuid4
from .project_files import AUTOSAVE_NAME, EXTENSION, OLD_EXTENSION

JOURNAL_NAME = "project-file-migrations.jsonl"
STOPPED = ("ScadaBench could not rename your saved project to the new file name ({old} to {new}). "
           "Nothing was deleted. Close other programs that use the data folder, then start ScadaBench "
           "again. Details are in logs/" + JOURNAL_NAME + ".")


def migrate(directory):
    """Rename every old-extension app file in a data folder; returns the journal records written."""
    directory = Path(directory)
    correlation = uuid4().hex
    records = []
    for old, new, protected in _plan(directory):
        record = _move(directory, old, new, correlation)
        if record is None:
            continue
        records.append(record)
        if protected and record["kind"] == "project_file_rename_failed":
            raise RuntimeError(STOPPED.format(old=old.name, new=new.name))
    return records


def _plan(directory):
    """(old path, new path, startup must stop if it cannot be renamed) for each app-owned file."""
    plan = [(directory / ("autosave" + OLD_EXTENSION), directory / AUTOSAVE_NAME, True)]
    recovery = directory / "recovery"
    if recovery.is_dir() and not recovery.is_symlink():
        try:
            names = sorted(entry.name for entry in os.scandir(recovery)
                           if entry.name.lower().endswith(OLD_EXTENSION))
        except OSError:
            names = []
        plan.extend((recovery / name, recovery / (name[:-len(OLD_EXTENSION)] + EXTENSION), False)
                    for name in names)
    return plan


def _move(directory, old, new, correlation):
    if not os.path.lexists(old):
        return None
    if old.is_symlink() or not old.is_file():
        return _write(directory, correlation, "project_file_left_alone", old, new, reason="not a regular file")
    if os.path.lexists(new):
        return _write(directory, correlation, "project_file_kept_both", old, new,
                      reason="the new name already exists; both files were kept")
    try:
        os.rename(old, new)
    except OSError as error:
        return _write(directory, correlation, "project_file_rename_failed", old, new, error=str(error))
    return _write(directory, correlation, "project_file_renamed", old, new)


def _write(directory, correlation, kind, old, new, **extra):
    record = {"kind": kind, "at": datetime.now(timezone.utc).isoformat(), "correlation_id": correlation,
              "from": old.relative_to(directory).as_posix(), "to": new.relative_to(directory).as_posix(), **extra}
    try:
        folder = directory / "logs"
        folder.mkdir(exist_ok=True)
        with (folder / JOURNAL_NAME).open("a", encoding="utf-8", newline="\n") as stream:
            stream.write(json.dumps(record, allow_nan=False, separators=(",", ":")) + "\n")
    except OSError:
        logging.getLogger(__name__).warning("Project file migration could not be journaled: %s", record)
    return record
