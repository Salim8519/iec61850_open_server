"""Single-file portable exe: keep projects and protocol workers beside the exe (ADR-0063).

The one-file ScadaBench-Portable exe unpacks itself into a new random temp folder on every
start. The native workers must not run from there: Windows firewall rules name the program
path, so a new path on each start would mean a new rule and a new UAC prompt. The workers are
copied once into ScadaBench-data/workers/<content hash>/native beside the exe (not
runtime/, where the app writes IED configuration files) and run from
there; the same build always gets the same path. Nothing is written outside the exe's folder.
"""
import hashlib
import os
from pathlib import Path
import shutil

DATA_FOLDER = "ScadaBench-data"
WORKER_FOLDER = "workers"


def is_single_file(executable_folder, resources):
    """True for the one-file exe: its resources are unpacked outside the exe's folder."""
    folder, unpacked = Path(executable_folder).resolve(), Path(resources).resolve()
    return folder != unpacked and folder not in unpacked.parents


def data_directory(executable_folder):
    return Path(executable_folder) / DATA_FOLDER


def _content_key(files):
    digest = hashlib.sha256()
    for path in files:
        digest.update(path.name.encode("utf-8") + b"\0" + hashlib.sha256(path.read_bytes()).digest())
    return digest.hexdigest()[:16]


def stage_native(source, data):
    """Copy the unpacked workers to a stable folder; return that folder.

    A finished copy is reused as it is. A copy is built under a temporary name and renamed,
    so an interrupted start never leaves a half-written worker folder in use."""
    files = sorted(Path(source).glob("*.exe"))
    if not files:
        raise RuntimeError(f"No native workers were unpacked in {source}")
    runtime = Path(data) / WORKER_FOLDER
    target = runtime / _content_key(files) / "native"
    if all((target / path.name).is_file() for path in files):
        return target
    staging = target.parent.with_name(target.parent.name + f".partial-{os.getpid()}")
    shutil.rmtree(staging, ignore_errors=True)
    (staging / "native").mkdir(parents=True)
    for path in files:
        shutil.copy2(path, staging / "native" / path.name)
    shutil.rmtree(target.parent, ignore_errors=True)
    try:
        os.replace(staging, target.parent)
    except OSError:
        shutil.rmtree(staging, ignore_errors=True)
        if not all((target / path.name).is_file() for path in files):
            raise
    return target
