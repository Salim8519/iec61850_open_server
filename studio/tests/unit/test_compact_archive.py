"""Persist engineering data once, not repeated source XML and derived signal tables."""
import io
import json
import zipfile
import pytest
from studio_app.contracts.errors import StudioError
from studio_app.projects.archive import encode, decode
from studio_app.projects.document import expand, CURRENT_FORMAT
from studio_app.projects.service import ProjectService


def test_sources_are_deduplicated_and_runtime_tables_excluded(tmp_path):
    projects = ProjectService(tmp_path)
    model = projects.add_template("Compact")[0]
    project = {**projects.project, "ieds": [model, {**model, "id": "b" * 32}]}
    payload = encode(project)
    with zipfile.ZipFile(io.BytesIO(payload)) as archive:
        stored = json.loads(archive.read("project.json"))
    assert stored["format"] == CURRENT_FORMAT
    assert len(stored["scl_sources"]) == 1
    assert all("signals" not in item and "cfg" not in item and "source" not in item for item in stored["ieds"])
    assert all(item["source"] == model["source"] for item in decode(payload)["ieds"])
    projects.restore(projects.export())
    assert len(projects.project["ieds"][0]["signals"]) == len(model["signals"])


def test_legacy_format_remains_readable(tmp_path):
    projects = ProjectService(tmp_path)
    projects.add_template("Legacy")
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("project.json", json.dumps({"format": 1, **projects.project}))
    projects.restore(stream.getvalue())
    assert projects.project["ieds"][0]["name"] == "Legacy"


def test_missing_and_corrupt_source_tables_are_rejected():
    with pytest.raises(StudioError, match="checksum"):
        expand({"format": 2, "scl_sources": {"wrong": "xml"}, "ieds": []})
    with pytest.raises(StudioError, match="missing"):
        expand({"format": 2, "scl_sources": {}, "ieds": [{"source_id": "missing"}]})


def test_size_limits_allow_bounded_legacy_recovery(monkeypatch):
    from studio_app.projects import archive as module
    monkeypatch.setattr(module, "MAX_UNPACKED", 1024)
    monkeypatch.setattr(module, "MAX_LEGACY_UNPACKED", 4096)

    def stored(version, size):
        stream = io.BytesIO()
        with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as archive:
            archive.writestr("project.json", json.dumps(
                {"format": version, "name": "x" * size, "ieds": [], "scl_sources": {}}))
        return stream.getvalue()

    assert decode(stored(1, 2000))["format"] == 1
    with pytest.raises(StudioError, match="Compact project"):
        decode(stored(2, 2000))
    with pytest.raises(StudioError, match="legacy recovery limit"):
        decode(stored(1, 5000))
    with pytest.raises(StudioError, match="nothing was written"):
        encode({"name": "x" * 2000, "ieds": []})
