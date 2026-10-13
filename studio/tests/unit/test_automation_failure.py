"""Unexpected automation failures pause explicitly rather than killing the clock."""
import time
from studio_app.projects.service import ProjectService
from studio_app.runtime.event_bus import EventBus
from studio_app.simulation.coordinator import Coordinator


def test_unexpected_cycle_error_is_visible_and_thread_survives(tmp_path):
    projects = ProjectService(tmp_path)
    events = EventBus()
    runtime = Coordinator(projects, events, tmp_path / "unused.exe", tmp_path)
    def fail(_elapsed):
        raise RuntimeError("injected cycle failure")
    runtime.automation.advance = fail
    try:
        runtime.state = "running"
        deadline = time.monotonic() + 2
        while runtime.state == "running" and time.monotonic() < deadline:
            time.sleep(0.01)
        assert runtime.state == "paused"
        assert runtime.thread.is_alive()
        recorded = events.after(0)
        assert any(event["kind"] == "automation_failed" and event["message"] == "injected cycle failure"
                   for event in recorded)
        assert any(event["kind"] == "runtime" and event["state"] == "paused" for event in recorded)
    finally:
        runtime.close()
