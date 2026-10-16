"""Packaged window lifecycle: one instance, show-again, close stops the app, handoff, windowed output."""
import os
import threading
from pathlib import Path
import pytest
from studio_app.launcher import app_window, single_instance, windowed

windows_only = pytest.mark.skipif(os.name != "nt", reason="Windows named events")


class FakeProcess:
    def __init__(self, alive=False, pid=4321):
        self.alive, self.pid = alive, pid

    def wait(self, timeout=None):
        return 0

    def poll(self):
        return None if self.alive else 0


def test_closing_the_window_stops_the_app(tmp_path):
    closed = []
    app_window.watch_window(FakeProcess(), lambda: closed.append(True), tmp_path).join(2)
    assert closed == [True]


def test_handoff_to_another_browser_keeps_the_app_running(tmp_path, monkeypatch):
    monkeypatch.setattr(app_window, "profile_in_use", lambda profile: True)
    closed = []
    app_window.watch_window(FakeProcess(), lambda: closed.append(True), tmp_path).join(2)
    assert closed == []


def test_profile_lock_held_by_a_browser_is_detected(tmp_path):
    assert not app_window.profile_in_use(tmp_path)
    lock = tmp_path / "lockfile"
    lock.write_text("stale")
    assert not app_window.profile_in_use(tmp_path) and not lock.exists()
    if os.name == "nt":
        import msvcrt
        with lock.open("w") as handle:  # Chromium holds it open; Windows then refuses deletion.
            msvcrt.locking(handle.fileno(), msvcrt.LK_NBLCK, 1)
            assert app_window.profile_in_use(tmp_path)
            msvcrt.locking(handle.fileno(), msvcrt.LK_UNLCK, 1)


def test_a_leftover_window_never_adopts_the_new_profile(tmp_path, monkeypatch):
    busy = {tmp_path / "browser-profile"}
    monkeypatch.setattr(app_window, "profile_in_use", lambda profile: Path(profile) in busy)
    assert app_window.choose_profile(tmp_path) == tmp_path / "browser-profile-2"


def window(tmp_path, monkeypatch, process):
    calls = []
    monkeypatch.setattr(app_window, "open_app_window", lambda url, profile: calls.append(("open", url)) or process)
    monkeypatch.setattr(app_window, "watch_window", lambda *args: None)
    monkeypatch.setattr(app_window.webbrowser, "open", lambda url: calls.append(("browser", url)))
    return app_window.AppWindow("http://127.0.0.1:1/#session=x", tmp_path, lambda: None).open(), calls


def test_show_focuses_the_running_window(tmp_path, monkeypatch):
    focused = []
    monkeypatch.setattr(app_window, "focus_process_window", lambda pid: focused.append(pid) or True)
    app, calls = window(tmp_path, monkeypatch, FakeProcess(alive=True))
    app.show()
    assert focused == [4321] and calls == [("open", app.url)]


def test_show_does_not_reopen_a_window_the_user_closed(tmp_path, monkeypatch):
    app, calls = window(tmp_path, monkeypatch, FakeProcess(alive=False))
    monkeypatch.setattr(app_window, "profile_in_use", lambda profile: False)
    app.show()
    assert calls == [("open", app.url)]


def test_show_reopens_the_default_browser_tab_without_edge(tmp_path, monkeypatch):
    app, calls = window(tmp_path, monkeypatch, None)
    app.show()
    assert calls == [("open", app.url), ("browser", app.url)]


def test_event_name_is_per_data_folder(tmp_path):
    first, second = single_instance.event_name(tmp_path / "a"), single_instance.event_name(tmp_path / "b")
    assert first.startswith(r"Local\ScadaBench-Show-") and first != second
    assert single_instance.event_name(tmp_path / "a") == first


@windows_only
def test_second_launch_asks_the_running_instance_to_show(tmp_path):
    assert not single_instance.request_show(tmp_path)
    shown = threading.Event()
    listener = single_instance.ShowListener(tmp_path, shown.set).start()
    try:
        assert single_instance.request_show(tmp_path)
        assert shown.wait(2)
    finally:
        listener.close()
    listener.thread.join(2)
    assert not listener.thread.is_alive()
    assert not single_instance.request_show(tmp_path)


def test_terminal_launches_print_instead_of_dialogs(capsys):
    windowed.tell_user("ScadaBench is already running", error=True)
    assert "already running" in capsys.readouterr().err


def test_second_launch_focuses_the_recorded_window_itself(tmp_path, monkeypatch):
    from studio_app.launcher import entrypoint, window_focus
    entrypoint.write_launcher_record(tmp_path, "http://127.0.0.1:1", window_pid=4321)
    focused, asked = [], []
    monkeypatch.setattr(window_focus, "focus_process_window", lambda pid: focused.append(pid) or True)
    monkeypatch.setattr(single_instance, "request_show", lambda directory: asked.append(directory) or True)
    assert entrypoint.show_running_instance(tmp_path, RuntimeError("busy")) == 0
    assert focused == [4321] and asked == []
    entrypoint.write_launcher_record(tmp_path, "http://127.0.0.1:1")
    assert entrypoint.show_running_instance(tmp_path, RuntimeError("busy")) == 0
    assert asked == [tmp_path]
