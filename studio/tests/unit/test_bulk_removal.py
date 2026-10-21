"""All-or-nothing deletion preserves the complete pre-delete engineering archive."""
from pathlib import Path
import pytest
from studio_app.projects.service import ProjectService
from studio_app.projects.bulk_removal import preview, remove


def test_bulk_impact_and_backup_restoration(tmp_path):
    projects = ProjectService(tmp_path)
    models = [projects.add_template(name)[0] for name in ("First", "Second", "Keep")]
    keys = [model["id"] for model in models[:2]]
    projects.test_panel([{"id": model["name"], "label": "Alarm", "ied_id": model["id"],
                         "ref": model["name"] + "CTRL/XCBR1.Loc.stVal"} for model in models])
    projects.diagram({"nodes": [{"id": "cb", "position": {"x": 1, "y": 2},
        "data": {"kind": "CB", "label": "CB", "state_mapping": {"mode": "direct",
                 "source": {"ied_id": keys[0], "ref": "FirstCTRL/XCBR1.Pos.stVal",
                            "report_ref": "FirstCTRL/LLN0.BR.Buffered"}}}}], "edges": []})
    candidate = preview(projects, keys)
    assert candidate["test_controls"] == 2 and candidate["bindings"] == 1
    result = remove(projects, keys, candidate["fingerprint"])
    assert [ied["name"] for ied in projects.project["ieds"]] == ["Keep"]
    assert len(projects.project["test_panel"]) == 1
    assert projects.project["diagram"]["nodes"][0]["data"]["state_mapping"] is None
    assert len(list((tmp_path / "recovery").glob("*.scadabench"))) == 1
    projects.restore(Path(result["recovery_file"]).read_bytes())
    assert len(projects.project["ieds"]) == 3 and len(projects.project["test_panel"]) == 3


def test_stale_preview_and_save_failure_do_not_remove(tmp_path, monkeypatch):
    projects = ProjectService(tmp_path)
    key = projects.add_template("Keep")[0]["id"]
    candidate = preview(projects, [key])
    projects.project["name"] = "Changed"
    with pytest.raises(ValueError, match="changed"):
        remove(projects, [key], candidate["fingerprint"])
    candidate = preview(projects, [key])
    original = projects.project
    def fail():
        raise OSError("Disk full")
    monkeypatch.setattr(projects, "save", fail)
    with pytest.raises(OSError):
        remove(projects, [key], candidate["fingerprint"])
    assert projects.project is original
    assert len(projects.project["ieds"]) == 1
    with pytest.raises(ValueError):
        preview(projects, [key, key])
