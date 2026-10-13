"""Live values keep device quality and are never invented (LIVE-07/09)."""
import pytest
from studio_app.acquisition.live_values import LiveValues
from studio_app.domain.live_value import convert


class FakeStates:
    def __init__(self):
        self.changed = []

    def publish_changed(self, keys):
        self.changed.extend(keys)


class FakeCoordinator:
    def __init__(self):
        import threading
        self.lock = threading.RLock()
        self.events = []
        self.equipment_states = FakeStates()
        signals = [{"ref": "D/XCBR1.Pos.stVal", "kind": "Dbpos", "value": 0,
                    "quality_ref": "D/XCBR1.Pos.q", "timestamp_ref": "D/XCBR1.Pos.t"},
                   {"ref": "D/XCBR1.Pos.q", "kind": "Quality", "value": 0},
                   {"ref": "D/XCBR1.Pos.t", "kind": "Timestamp", "value": 0}]
        self.ied_indexes = {"ied": {item["ref"]: item for item in signals}}

    def emit(self, kind, **payload):
        self.events.append({"kind": kind, **payload})


def test_device_quality_and_source_are_kept():
    coordinator = FakeCoordinator()
    result = LiveValues(coordinator).apply("ied", {
        "D/XCBR1.Pos.stVal": {"bits": 2, "le": 1, "be": 2},
        "D/XCBR1.Pos.q": {"bits": 13, "le": 64, "be": 2},
        "D/XCBR1.Pos.t": 1791921707689}, "live_report")
    assert result == {"applied": 3, "errors": {}}
    signal_events = [event for event in coordinator.events if event["kind"] == "signal"]
    assert len(signal_events) == 1
    event = signal_events[0]
    assert event["value"] == 2 and event["quality"] == 64 and event["source_timestamp"] == 1791921707689
    assert event["source"] == "live_report" and event["live_state"] == "fresh"
    with pytest.raises(ValueError):
        LiveValues(coordinator).apply("ied", {}, "manual")


def test_read_errors_are_not_values():
    coordinator = FakeCoordinator()
    result = LiveValues(coordinator).apply("ied", {"D/XCBR1.Pos.stVal": {"access_error": 10},
                                                   "D/XCBR1.Pos.t": "yesterday"}, "live_poll")
    assert result["applied"] == 0 and set(result["errors"]) == {"D/XCBR1.Pos.stVal", "D/XCBR1.Pos.t"}
    assert coordinator.ied_indexes["ied"]["D/XCBR1.Pos.stVal"]["value"] == 0
    assert [event["kind"] for event in coordinator.events] == ["live_read_error", "live_read_error"]
    with pytest.raises(ValueError, match="no value"):
        convert("BOOLEAN", None)
    with pytest.raises(ValueError):
        convert("BOOLEAN", 1)


def test_connection_loss_marks_values_stale():
    coordinator = FakeCoordinator()
    values = LiveValues(coordinator)
    values.apply("ied", {"D/XCBR1.Pos.stVal": {"bits": 2, "le": 1, "be": 2}}, "live_poll")
    values.mark_stale("ied", "Connection to the device was lost")
    signal = coordinator.ied_indexes["ied"]["D/XCBR1.Pos.stVal"]
    assert signal["live_state"] == "stale" and signal["value"] == 2
    assert coordinator.events[-1] == {"kind": "live_stale", "ied_id": "ied",
                                      "message": "Connection to the device was lost"}
