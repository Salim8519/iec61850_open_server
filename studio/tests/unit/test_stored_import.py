"""Restoring a project parses each shared SCL source once and keeps every IED's model."""
from pathlib import Path

from studio_app.projects import stored_import
from studio_app.projects.service import ProjectService
from studio_app.scl.importer import import_scl

ROOT = Path(__file__).resolve().parents[3]
SOURCE = (ROOT / "scd/open_substation.scd").read_bytes()
NAMES = ["IED2_PTOC", "IED1_XCBR"]


def test_shared_source_is_imported_once_with_identical_models(tmp_path, monkeypatch):
    projects = ProjectService(tmp_path)
    projects.add_scl(SOURCE, "station.scd", NAMES)
    before = projects.snapshot()
    calls = []

    def counting(source, filename, names):
        calls.append(list(names))
        return import_scl(source, filename, names)
    monkeypatch.setattr(stored_import, "import_scl", counting)
    projects.restore(projects.export())
    after = projects.snapshot()
    assert len(calls) == 1 and sorted(calls[0]) == sorted(NAMES)
    assert [ied["id"] for ied in after["ieds"]] == [ied["id"] for ied in before["ieds"]]
    for old, new in zip(before["ieds"], after["ieds"]):
        assert new == old


def test_each_restored_ied_owns_its_validation_report(tmp_path):
    projects = ProjectService(tmp_path)
    projects.add_scl(SOURCE, "station.scd", NAMES)
    projects.restore(projects.export())
    first, second = projects.project["ieds"]
    assert first["validation"] == second["validation"]
    assert first["validation"] is not second["validation"]
