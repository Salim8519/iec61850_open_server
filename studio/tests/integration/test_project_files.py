"""Project files are .scadabench; an old .subsim file is refused (ADR-0096). Runs on both engines."""
import io
import json
from pathlib import Path
from studio_app.app_factory import create_app
from studio_app.projects.service import ProjectService
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]
MESSAGE = "This is an old project file (.subsim). ScadaBench now uses .scadabench project files."


def start(folder):
    settings = Settings(folder, ROOT / "studio/native/build/Release/studio_ied.exe", folder)
    app = create_app(settings, "files-test")
    assert app.extensions["startup"].wait(60)
    client = app.test_client()
    assert client.post("/api/v1/session", json={"token": "files-test"}).status_code == 200
    return app, client


def journal(folder):
    path = folder / "logs" / "project-file-migrations.jsonl"
    return [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()] if path.exists() else []


def old_data_folder(folder, name="Legacy"):
    """A data folder as an older ScadaBench left it: autosave.subsim and a recovery backup."""
    ProjectService(folder).add_template(name)
    (folder / "autosave.scadabench").rename(folder / "autosave.subsim")
    (folder / "recovery").mkdir()
    (folder / "recovery" / "before-ied-delete-0001.subsim").write_bytes(b"backup bytes")
    return (folder / "autosave.subsim").read_bytes()


def test_save_download_and_open_round_trip_use_scadabench(tmp_path):
    app, client = start(tmp_path)
    try:
        assert client.post("/api/v1/ieds/template", json={"name": "Feeder01"}).status_code == 200
        assert client.post("/api/v1/project/save").status_code == 200
        assert (tmp_path / "autosave.scadabench").is_file() and not (tmp_path / "autosave.subsim").exists()
        archive = client.get("/api/v1/project/download")
        assert archive.status_code == 200
        assert "filename=substation.scadabench" in archive.headers["Content-Disposition"]
        opened = client.post("/api/v1/project/open", data={"file": (io.BytesIO(archive.data), "mine.scadabench")})
        assert opened.status_code == 200
        assert [ied["name"] for ied in opened.json["ieds"]] == ["Feeder01"]
    finally:
        app.extensions["runtime"].close()


def test_old_subsim_file_is_refused_and_the_project_is_untouched(tmp_path):
    app, client = start(tmp_path)
    try:
        client.post("/api/v1/ieds/template", json={"name": "Feeder01"})
        archive = client.get("/api/v1/project/download").data
        for name in ("old.subsim", "OLD.SubSim"):
            refused = client.post("/api/v1/project/open", data={"file": (io.BytesIO(archive), name)})
            assert refused.status_code == 400
            assert refused.json["error"] == MESSAGE and refused.json["code"] == "old_project_file"
        assert [ied["name"] for ied in client.get("/api/v1/project").json["ieds"]] == ["Feeder01"]
    finally:
        app.extensions["runtime"].close()


def test_old_app_files_are_renamed_on_startup_without_loss(tmp_path):
    original = old_data_folder(tmp_path)
    app, client = start(tmp_path)
    try:
        assert [ied["name"] for ied in client.get("/api/v1/project").json["ieds"]] == ["Legacy"]
        assert not (tmp_path / "autosave.subsim").exists()
        assert (tmp_path / "autosave.scadabench").read_bytes() == original
        assert not (tmp_path / "recovery" / "before-ied-delete-0001.subsim").exists()
        assert (tmp_path / "recovery" / "before-ied-delete-0001.scadabench").read_bytes() == b"backup bytes"
        written = journal(tmp_path)
        assert sorted((line["kind"], line["from"], line["to"]) for line in written) == [
            ("project_file_renamed", "autosave.subsim", "autosave.scadabench"),
            ("project_file_renamed", "recovery/before-ied-delete-0001.subsim",
             "recovery/before-ied-delete-0001.scadabench")]
        assert len({line["correlation_id"] for line in written}) == 1
    finally:
        app.extensions["runtime"].close()


def test_both_names_present_keeps_both_and_uses_the_new_one(tmp_path):
    original = old_data_folder(tmp_path, "Older")
    ProjectService(tmp_path / "other").add_template("Newer")
    (tmp_path / "autosave.scadabench").write_bytes((tmp_path / "other" / "autosave.scadabench").read_bytes())
    (tmp_path / "recovery" / "before-ied-delete-0001.scadabench").write_bytes(b"already here")
    app, client = start(tmp_path)
    try:
        assert [ied["name"] for ied in client.get("/api/v1/project").json["ieds"]] == ["Newer"]
        assert (tmp_path / "autosave.subsim").read_bytes() == original
        assert (tmp_path / "recovery" / "before-ied-delete-0001.subsim").read_bytes() == b"backup bytes"
        assert (tmp_path / "recovery" / "before-ied-delete-0001.scadabench").read_bytes() == b"already here"
        assert sorted(line["kind"] for line in journal(tmp_path)) == ["project_file_kept_both"] * 2
    finally:
        app.extensions["runtime"].close()
