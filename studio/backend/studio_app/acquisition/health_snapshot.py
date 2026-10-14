"""Collect the live sessions and GOOSE listener state into one station health result."""
import copy
from datetime import datetime, timezone
from studio_app.domain.mode_badges import mode_badges, simulation_sources
from studio_app.domain.station_health import build_health


def station_health(projects, coordinator):
    with projects.lock:
        ieds = copy.deepcopy(projects.project["ieds"])
        diagram = copy.deepcopy(projects.project["diagram"])
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
