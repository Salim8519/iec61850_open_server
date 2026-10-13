"""Recovery must retain engineering identity and never auto-start or erase evidence."""
from pathlib import Path
import pytest
from studio_app.app_factory import create_app
from studio_app.projects.service import ProjectService
from studio_app.projects.startup_recovery import recover_startup
from studio_app.settings import Settings


def test_startup_recovers_engineering_without_writing_or_running(tmp_path):
    saved = ProjectService(tmp_path)
    ied = saved.add_template("Recovered")[0]
    saved.source(ied["id"], "RecoveredCTRL/GGIO1.AnIn1.mag.f",
                 {"mode": "constant", "offset": 42, "amplitude": 1, "period": 1})
    saved.diagram({"nodes": [{"id": "cb", "position": {"x": 100, "y": 200},
        "data": {"kind": "CB", "label": "Recovered CB", "state_mapping": {"mode": "direct",
                 "source": {"ied_id": ied["id"],
                            "ref": "RecoveredCTRL/XCBR1.Pos.stVal"}}}}], "edges": []})
    saved.scenarios([{"id": "sequence", "name": "Saved sequence", "enabled": True,
        "steps": [{"id": "set", "at": 0, "ied_id": ied["id"],
                   "ref": "RecoveredCTRL/GGIO1.AnIn1.mag.f", "value": 15}]}])
    payload = (tmp_path / "autosave.subsim").read_bytes()
    app = create_app(Settings(tmp_path, tmp_path / "does-not-exist.exe", tmp_path), "recovery-test")
    try:
        restored = app.extensions["projects"].snapshot()
        assert restored["id"] == saved.project["id"]
        assert restored["ieds"][0]["id"] == ied["id"]
        assert restored["sources"] == saved.snapshot()["sources"]
        assert restored["diagram"] == saved.snapshot()["diagram"]
        assert restored["scenarios"] == saved.snapshot()["scenarios"]
        assert restored["ieds"][0]["state"] == "stopped"
        assert app.extensions["runtime"].state == "stopped"
        assert app.extensions["runtime"].supervisor.workers == {}
        assert (tmp_path / "autosave.subsim").read_bytes() == payload
        client = app.test_client()
        client.post("/api/v1/session", json={"token": "recovery-test"})
        assert client.get("/api/v1/project").json["startup_recovery"]["status"] == "restored"
    finally:
        app.extensions["runtime"].close()


def test_corrupt_autosave_is_preserved_before_new_work(tmp_path):
    original = b"interrupted or damaged archive"
    (tmp_path / "autosave.subsim").write_bytes(original)
    projects = ProjectService(tmp_path)
    result = recover_startup(projects)
    assert result["status"] == "unreadable"
    backup = Path(result["backup"])
    assert backup.read_bytes() == original
    assert (tmp_path / "autosave.subsim").read_bytes() == original
    assert projects.project["ieds"] == []
    projects.add_template("NewWork")
    assert backup.read_bytes() == original


def test_recovery_backup_failure_stops_startup(tmp_path, monkeypatch):
    (tmp_path / "autosave.subsim").write_bytes(b"damaged")
    def fail(*_args):
        raise PermissionError("injected backup failure")
    monkeypatch.setattr("studio_app.projects.startup_recovery.shutil.copyfile", fail)
    with pytest.raises(RuntimeError, match="startup stopped"):
        recover_startup(ProjectService(tmp_path))
    assert (tmp_path / "autosave.subsim").read_bytes() == b"damaged"


def test_project_restore_save_failure_preserves_current_project(tmp_path, monkeypatch):
    projects = ProjectService(tmp_path)
    previous = projects.project
    other = ProjectService(tmp_path / "other")
    other.add_template("Other")
    def fail():
        raise PermissionError("injected save failure")
    monkeypatch.setattr(projects, "save", fail)
    with pytest.raises(PermissionError):
        projects.restore(other.export())
    assert projects.project is previous
