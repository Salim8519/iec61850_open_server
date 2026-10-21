"""Old app-owned .subsim files become .scadabench at startup and are never lost (ADR-0096)."""
import json
import pytest
from studio_app.projects import extension_migration
from studio_app.projects.extension_migration import migrate
from studio_app.projects.project_files import OLD_PROJECT_MESSAGE, refuse_old_project
from studio_app.projects.service import ProjectService
from studio_app.contracts.errors import StudioError


def lines(folder):
    text = (folder / "logs" / "project-file-migrations.jsonl").read_text(encoding="utf-8")
    return [json.loads(line) for line in text.splitlines()]


def test_nothing_to_migrate_writes_nothing(tmp_path):
    assert migrate(tmp_path) == []
    assert not (tmp_path / "logs").exists()


def test_autosave_and_recovery_files_are_renamed_with_their_bytes(tmp_path):
    (tmp_path / "autosave.subsim").write_bytes(b"project")
    (tmp_path / "recovery").mkdir()
    (tmp_path / "recovery" / "unreadable-autosave-ab.SUBSIM").write_bytes(b"damaged")
    (tmp_path / "recovery" / "notes.txt").write_bytes(b"leave me")
    records = migrate(tmp_path)
    assert [item["kind"] for item in records] == ["project_file_renamed"] * 2
    assert (tmp_path / "autosave.scadabench").read_bytes() == b"project"
    assert (tmp_path / "recovery" / "unreadable-autosave-ab.scadabench").read_bytes() == b"damaged"
    assert (tmp_path / "recovery" / "notes.txt").read_bytes() == b"leave me"
    assert sorted(path.name for path in tmp_path.iterdir()) == ["autosave.scadabench", "logs", "recovery"]
    assert lines(tmp_path) == records


def test_running_twice_changes_nothing_the_second_time(tmp_path):
    (tmp_path / "autosave.subsim").write_bytes(b"project")
    migrate(tmp_path)
    assert migrate(tmp_path) == []
    assert len(lines(tmp_path)) == 1


def test_an_existing_new_name_is_never_overwritten(tmp_path):
    (tmp_path / "autosave.subsim").write_bytes(b"old")
    (tmp_path / "autosave.scadabench").write_bytes(b"new")
    records = migrate(tmp_path)
    assert [item["kind"] for item in records] == ["project_file_kept_both"]
    assert (tmp_path / "autosave.subsim").read_bytes() == b"old"
    assert (tmp_path / "autosave.scadabench").read_bytes() == b"new"


def test_a_folder_with_the_old_name_is_left_alone(tmp_path):
    (tmp_path / "autosave.subsim").mkdir()
    assert [item["kind"] for item in migrate(tmp_path)] == ["project_file_left_alone"]
    assert (tmp_path / "autosave.subsim").is_dir() and not (tmp_path / "autosave.scadabench").exists()


def test_an_autosave_that_cannot_be_renamed_stops_startup_and_stays_put(tmp_path, monkeypatch):
    (tmp_path / "autosave.subsim").write_bytes(b"project")

    def fail(*_args):
        raise PermissionError("injected rename failure")
    monkeypatch.setattr(extension_migration.os, "rename", fail)
    with pytest.raises(RuntimeError, match="Nothing was deleted"):
        migrate(tmp_path)
    assert (tmp_path / "autosave.subsim").read_bytes() == b"project"
    assert [item["kind"] for item in lines(tmp_path)] == ["project_file_rename_failed"]


def test_a_recovery_backup_that_cannot_be_renamed_does_not_stop_startup(tmp_path, monkeypatch):
    (tmp_path / "recovery").mkdir()
    (tmp_path / "recovery" / "a.subsim").write_bytes(b"backup")

    def fail(*_args):
        raise PermissionError("injected rename failure")
    monkeypatch.setattr(extension_migration.os, "rename", fail)
    assert [item["kind"] for item in migrate(tmp_path)] == ["project_file_rename_failed"]
    assert (tmp_path / "recovery" / "a.subsim").read_bytes() == b"backup"


def test_a_project_service_migrates_before_it_reads_anything(tmp_path):
    saved = ProjectService(tmp_path)
    saved.add_template("Kept")
    (tmp_path / "autosave.scadabench").rename(tmp_path / "autosave.subsim")
    ProjectService(tmp_path)
    assert (tmp_path / "autosave.scadabench").is_file() and not (tmp_path / "autosave.subsim").exists()


def test_only_the_old_extension_is_refused():
    for name in ("a.subsim", "A.SUBSIM", "C:\\projects\\station.SubSim"):
        with pytest.raises(StudioError) as refused:
            refuse_old_project(name)
        assert str(refused.value) == OLD_PROJECT_MESSAGE and refused.value.code == "old_project_file"
    for name in ("a.scadabench", "a.zip", "subsim", "", None):
        refuse_old_project(name)
