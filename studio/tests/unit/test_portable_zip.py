"""The portable zip is the sealed app folder plus an empty data folder, never user data or source."""
import importlib.util
from pathlib import Path
import zipfile

import pytest

TOOL = Path(__file__).resolve().parents[2] / "tools" / "make_portable.py"
spec = importlib.util.spec_from_file_location("make_portable", TOOL)
make_portable = importlib.util.module_from_spec(spec)
spec.loader.exec_module(make_portable)


def app_folder(root):
    folder = root / "ScadaBench"
    (folder / "_internal" / "native").mkdir(parents=True)
    (folder / "ScadaBench.exe").write_bytes(b"exe")
    (folder / "_internal" / "native" / "studio_ied.exe").write_bytes(b"worker")
    return folder


def test_portable_zip_holds_the_app_an_empty_data_folder_and_a_readme(tmp_path):
    folder = app_folder(tmp_path)
    size = make_portable.make(folder, tmp_path / "portable.zip")
    names = zipfile.ZipFile(tmp_path / "portable.zip").namelist()
    assert size > 0 and "ScadaBench/ScadaBench.exe" in names and "ScadaBench/data/" in names
    assert "ScadaBench/_internal/native/studio_ied.exe" in names and "ScadaBench/PORTABLE.txt" in names


def test_portable_zip_refuses_user_data_source_or_a_missing_build(tmp_path):
    with pytest.raises(SystemExit, match="compile_protected"):
        make_portable.make(tmp_path / "nothing", tmp_path / "a.zip")
    folder = app_folder(tmp_path)
    (folder / "data").mkdir()
    (folder / "data" / "autosave.subsim").write_text("customer", encoding="utf-8")
    with pytest.raises(SystemExit, match="user data"):
        make_portable.make(folder, tmp_path / "b.zip")
    (folder / "data" / "autosave.subsim").unlink()
    (folder / "_internal" / "studio_app").mkdir()
    (folder / "_internal" / "studio_app" / "service.py").write_text("source", encoding="utf-8")
    with pytest.raises(SystemExit, match="Source"):
        make_portable.make(folder, tmp_path / "c.zip")
