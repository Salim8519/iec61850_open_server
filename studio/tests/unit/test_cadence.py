"""A late scheduler must not silently reset deadlines or replay unlimited work."""
import pytest
from studio_app.simulation.cadence import Cadence


def test_early_wake_and_regular_deadlines():
    cadence = Cadence(0)
    assert cadence.take(0.05) is None
    assert cadence.take(0.1) == {"lateness_seconds": 0, "skipped_ticks": 0}
    assert cadence.deadline == pytest.approx(0.2)


def test_transient_stall_keeps_original_cadence():
    cadence = Cadence(0)
    results = [cadence.take(0.35) for _ in range(3)]
    assert [result["skipped_ticks"] for result in results] == [0, 0, 0]
    assert results[0]["lateness_seconds"] == pytest.approx(0.25)
    assert cadence.take(0.35) is None
    assert cadence.deadline == pytest.approx(0.4)


def test_long_stall_bounds_backlog_and_counts_every_skipped_tick():
    cadence = Cadence(0)
    first = cadence.take(2.05)
    assert first["skipped_ticks"] == 15
    assert first["lateness_seconds"] == pytest.approx(1.95)
    remaining = [cadence.take(2.05) for _ in range(4)]
    assert all(result is not None and result["skipped_ticks"] == 0 for result in remaining)
    assert cadence.take(2.05) is None
    assert cadence.deadline == pytest.approx(2.1)


def test_resume_does_not_replay_paused_time():
    cadence = Cadence(0)
    cadence.reset(900)
    assert cadence.take(900) is None
    assert cadence.take(900.1)["skipped_ticks"] == 0
