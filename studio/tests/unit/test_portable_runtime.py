"""The single-file portable exe runs its workers from a stable folder beside it (ADR-0063).

A one-file exe unpacks into a new random temp folder each start; firewall rules name the
program path, so workers run from there would need a new rule and UAC prompt every start."""
from pathlib import Path

from studio_app.launcher import portable_runtime
from studio_app.settings import Settings


def unpacked(root, content=b"worker"):
    native = root / "_MEI1234" / "native"
    native.mkdir(parents=True)
    (native / "studio_ied.exe").write_bytes(content)
    (native / "studio_live.exe").write_bytes(b"live")
    return native


def test_one_file_is_told_apart_from_the_installed_folder(tmp_path):
    exe_folder = tmp_path / "Downloads"
    exe_folder.mkdir()
    assert portable_runtime.is_single_file(exe_folder, tmp_path / "Temp" / "_MEI99")
    assert not portable_runtime.is_single_file(exe_folder, exe_folder / "_internal")


def test_workers_get_the_same_stable_path_on_every_start(tmp_path):
    data = tmp_path / "Downloads" / portable_runtime.DATA_FOLDER
    first = portable_runtime.stage_native(unpacked(tmp_path / "start1"), data)
    second = portable_runtime.stage_native(unpacked(tmp_path / "start2"), data)
    assert first == second and first.is_relative_to(data)
    assert (first / "studio_ied.exe").read_bytes() == b"worker"
    assert not list(data.glob("workers/*.partial-*"))


def test_a_new_build_gets_its_own_worker_folder(tmp_path):
    data = tmp_path / portable_runtime.DATA_FOLDER
    old = portable_runtime.stage_native(unpacked(tmp_path / "a", b"0.1.9"), data)
    new = portable_runtime.stage_native(unpacked(tmp_path / "b", b"0.2.0"), data)
    assert old != new and (new / "studio_ied.exe").read_bytes() == b"0.2.0"


def test_a_half_written_worker_folder_is_completed(tmp_path):
    data = tmp_path / portable_runtime.DATA_FOLDER
    target = portable_runtime.stage_native(unpacked(tmp_path / "a"), data)
    (target / "studio_live.exe").unlink()
    again = portable_runtime.stage_native(unpacked(tmp_path / "b"), data)
    assert again == target and (target / "studio_live.exe").is_file()


def test_frozen_one_file_settings_keep_everything_beside_the_exe(tmp_path, monkeypatch):
    exe_folder = tmp_path / "USB"
    exe_folder.mkdir()
    meipass = tmp_path / "Temp" / "_MEI1234"
    unpacked(tmp_path / "Temp")
    monkeypatch.setattr("sys.frozen", True, raising=False)
    monkeypatch.setattr("sys._MEIPASS", str(meipass), raising=False)
    monkeypatch.setattr("sys.executable", str(exe_folder / "ScadaBench-Portable-0.1.9.exe"))
    settings = Settings.defaults()
    assert settings.directory == exe_folder / portable_runtime.DATA_FOLDER
    assert Path(settings.native_executable).is_relative_to(exe_folder)
    assert settings.native_executable.name == "studio_ied.exe" and settings.native_executable.is_file()


def test_workers_never_share_the_folder_where_ied_configurations_are_written(tmp_path):
    """The coordinator writes <data>/runtime/<ied>.cfg; seen in the 0.1.9 portable test."""
    target = portable_runtime.stage_native(unpacked(tmp_path / "a"), tmp_path / "data")
    assert "runtime" not in target.relative_to(tmp_path / "data").parts
