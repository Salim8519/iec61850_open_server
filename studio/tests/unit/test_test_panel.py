"""Saved test controls preserve exact mappings without requiring drawing symbols."""
import io
import json
import zipfile
import pytest
from studio_app.projects.service import ProjectService
from studio_app.projects.archive import decode
from studio_app.projects.document import CURRENT_FORMAT


def entry(ied, reference=None):
    return {"id": "test-point", "label": "Breaker local mode", "ied_id": ied["id"],
            "ref": reference or ied["name"] + "CTRL/XCBR1.Loc.stVal", "report_ref": None}


def test_independent_control_roundtrip_and_version(tmp_path):
    projects = ProjectService(tmp_path)
    ied = projects.add_template("Panel")[0]
    projects.test_panel([entry(ied)])
    payload = projects.export()
    assert decode(payload)["format"] == CURRENT_FORMAT
    projects.restore(payload)
    assert projects.project["diagram"]["nodes"] == []
    assert projects.project["test_panel"][0]["duration_ms"] == 1000
    assert projects.project["test_panel"][0]["ref"] == "PanelCTRL/XCBR1.Loc.stVal"


def test_report_mismatch_readonly_and_duplicate_rejected(tmp_path):
    projects = ProjectService(tmp_path)
    ied = projects.add_template("Panel")[0]
    with pytest.raises(ValueError):
        projects.test_panel([{**entry(ied), "report_ref": "PanelCTRL/LLN0.BR.Buffered"}])
    with pytest.raises(ValueError):
        projects.test_panel([entry(ied, "PanelCTRL/XCBR1.Pos.ctlModel")])
    with pytest.raises(ValueError):
        projects.test_panel([entry(ied), entry(ied)])
    with pytest.raises(ValueError):
        projects.test_panel([{**entry(ied), "duration_ms": 99}])
    assert projects.project["test_panel"] == []


def test_format_two_migration_and_failed_save_rollback(tmp_path, monkeypatch):
    projects = ProjectService(tmp_path)
    ied = projects.add_template("Panel")[0]
    with zipfile.ZipFile(io.BytesIO(projects.export())) as archive:
        document = json.loads(archive.read("project.json"))
    document["format"] = 2
    document.pop("test_panel")
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("project.json", json.dumps(document))
    projects.restore(stream.getvalue())
    assert projects.project["test_panel"] == []
    def fail():
        raise OSError("Disk full")
    monkeypatch.setattr(projects, "save", fail)
    with pytest.raises(OSError):
        projects.test_panel([entry(ied)])
    assert projects.project["test_panel"] == []
