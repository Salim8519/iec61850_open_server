"""Collect the live sessions and GOOSE listener state into one station health result."""
import copy
import threading
from datetime import datetime, timezone
from studio_app.domain.mode_badges import mode_badges, simulation_sources
from studio_app.domain.station_health import build_health
from studio_app.domain.test_mode import mode_signals

# Health never reads the SCL text or the full point list. Copying 90,000 points on every
# poll held the project lock for over a second and kept the backend busy (change 0110).
_NOT_READ = {"source", "cfg", "signals"}
_mode_cache = {}  # IED id -> (the IED's signal list, its Mod/Beh/Sim points)
_mode_lock = threading.Lock()


def _mode_points(ied):
    """Point references and FCs never change in place; a new model brings a new list object."""
    signals = ied["signals"]
    with _mode_lock:
        cached = _mode_cache.get(ied["id"])
        if cached is None or cached[0] is not signals:
            cached = (signals, mode_signals(signals))
            _mode_cache[ied["id"]] = cached
    return cached[1]


def _forget_removed(ieds):
    keep = {ied["id"] for ied in ieds}
    with _mode_lock:
        for key in [key for key in _mode_cache if key not in keep]:
            del _mode_cache[key]


def health_view(ied):
    """Copy of what health reads from one IED, with only the mode points of its model."""
    view = {key: copy.deepcopy(value) for key, value in ied.items() if key not in _NOT_READ}
    view["signals"] = _mode_points(ied)
    return view


def station_health(projects, coordinator):
    with projects.lock:
        ieds = [health_view(ied) for ied in projects.project["ieds"]]
        diagram = copy.deepcopy(projects.project["diagram"])
    _forget_removed(ieds)
    running = coordinator.state != "stopped"
    live = coordinator.live
    sessions = {key: session.status for key, session in list(live.sessions.items())} if running else {}
    goose = live.goose.snapshot() if running else {}
    sv = live.sv.snapshot() if running else {}
    result = build_health(ieds, running, sessions, goose, datetime.now(timezone.utc))
    result["modes"] = mode_badges(ieds, sessions, diagram, running)
    result["simulation_sources"] = simulation_sources(goose, sv) if running else []
    return result


def refresh(coordinator):
    """Read the report-block, GOOSE-block and mode settings again on each existing session."""
    failures = {}
    for key, session in list(coordinator.live.sessions.items()):
        try:
            session.refresh_health()
        except (ValueError, RuntimeError, TimeoutError) as error:
            failures[key] = str(error)
    return failures
