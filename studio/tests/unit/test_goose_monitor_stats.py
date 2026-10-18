"""GOOSE retransmissions keep the page's sqNum current (change 0091).

Found between two PCs: a VM published sqNum 4, 5 ... (10 s heartbeat) while the page kept
showing sqNum 0, because only a state change carried sqNum to the backend."""
from types import SimpleNamespace

from studio_app.acquisition.goose_monitor import GooseMonitor


def test_stats_update_sq_num_without_a_state_change():
    monitor = GooseMonitor(SimpleNamespace(coordinator=None))
    monitor.watches = {"B/LLN0$GO$g": {"key": "B/LLN0$GO$g", "state": "receiving", "st_num": 2, "sq_num": 0,
                                       "frames": 3}}
    monitor._event({"event": "goose_stats", "watches": [{"key": "B/LLN0$GO$g", "frames": 5, "sq_num": 2,
                                                         "age_ms": 120}]})
    watch = monitor.watches["B/LLN0$GO$g"]
    assert (watch["frames"], watch["sq_num"], watch["st_num"], watch["age_ms"]) == (5, 2, 2, 120)


def test_stats_without_sq_num_keep_the_last_known_value():
    monitor = GooseMonitor(SimpleNamespace(coordinator=None))
    monitor.watches = {"k": {"key": "k", "sq_num": 7, "frames": 1}}
    monitor._event({"event": "goose_stats", "watches": [{"key": "k", "frames": 2, "age_ms": 5}]})
    assert monitor.watches["k"]["sq_num"] == 7
