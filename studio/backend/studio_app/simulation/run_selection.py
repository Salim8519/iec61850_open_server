"""Run or stop single IEDs while the rest of the run continues (ADR-0067).

The coordinator keeps one run; this module adds IEDs to it or removes them from it,
keeping the signal index, sources, sample locks, workers and live sessions consistent.
"""
import threading
from studio_app.projects.ied_settings import endpoint_conflicts
from .bindings import publish_bindings
from .control_compatibility import profiles as compatibility_profiles
from .initial_state import apply_initial_states
from .test_mode_apply import apply_test_modes


def runnable(project, selection=None):
    """IEDs a Run includes: every IED not set to Off, or only the selected ones."""
    ieds = [ied for ied in project["ieds"] if ied.get("role", "simulated") != "off"]
    if selection is None:
        return ieds
    by_id = {ied["id"]: ied for ied in project["ieds"]}
    for ied_id in selection:
        ied = by_id.get(ied_id)
        if ied is None:
            raise ValueError("Unknown IED")
        if ied.get("role", "simulated") == "off":
            raise ValueError(f"{ied['name']} is set to Off. Choose 'Simulate' or 'Connect to real device' first.")
    return [ied for ied in ieds if ied["id"] in selection]


def simulated_of(ieds):
    return [ied for ied in ieds if ied.get("role", "simulated") == "simulated"]


def running_ieds(coordinator):
    return [ied for ied in coordinator.projects.project["ieds"] if ied["id"] in coordinator.active]


def add_ieds(coordinator, selection):
    """Start more IEDs inside the current run without touching the running ones."""
    project = coordinator.projects.project
    new = [ied for ied in runnable(project, selection) if ied["id"] not in coordinator.active]
    if not new:
        raise ValueError("This IED is already running")
    current = running_ieds(coordinator)
    conflicts = endpoint_conflicts([*current, *new], project["network"])
    if conflicts:
        raise ValueError(conflicts[0])
    simulated, real = simulated_of(new), [ied for ied in new if ied.get("role") == "real"]
    ids = {ied["id"] for ied in new}
    with coordinator.lock:
        for ied in new:
            coordinator.ied_indexes[ied["id"]] = {signal["ref"]: signal for signal in ied["signals"]}
            coordinator.index.update({(ied["id"], signal["ref"]): signal for signal in ied["signals"]})
        sim_ids = {ied["id"] for ied in simulated}
        coordinator.sources.update({(item["ied_id"], item["ref"]): dict(item["spec"], start_elapsed=coordinator.elapsed)
                                    for item in project["sources"] if item["ied_id"] in sim_ids})
        coordinator.sample_writer.locks.update({ied_id: threading.RLock() for ied_id in sim_ids})
        coordinator.active |= ids
    try:
        if simulated:
            coordinator.supervisor.add(simulated, compatibility_profiles(project["diagram"]), project["network"],
                                       simulated_of(current))
        coordinator.live.connect(project, real)
    except Exception:
        _forget(coordinator, new)
        raise
    publish_bindings(coordinator, ids)
    apply_initial_states(coordinator, ids)
    apply_test_modes(coordinator, ids)
    coordinator.equipment_states.publish_all()
    for ied in new:
        coordinator.emit("ied_runtime", ied_id=ied["id"], state="started", role=ied.get("role", "simulated"))
    coordinator.emit("runtime", state=coordinator.state, running=sorted(coordinator.active))


def remove_ieds(coordinator, selection):
    """Stop some IEDs; the others keep their workers, sessions and clients."""
    gone = [ied for ied in coordinator.projects.project["ieds"] if ied["id"] in selection and ied["id"] in coordinator.active]
    if not gone:
        raise ValueError("This IED is not running")
    for ied in gone:
        coordinator.pulses.cancel(ied["id"], reason="IED stopped")
    _forget(coordinator, gone)
    coordinator.equipment_states.publish_all()
    for ied in gone:
        coordinator.emit("ied_runtime", ied_id=ied["id"], state="stopped", role=ied.get("role", "simulated"))
    coordinator.emit("runtime", state=coordinator.state, running=sorted(coordinator.active))


def _forget(coordinator, ieds):
    ids = {ied["id"] for ied in ieds}
    coordinator.supervisor.stop_some(simulated_of(ieds))
    for ied in ieds:
        if ied.get("role") == "real":
            coordinator.live.disconnect(ied["id"])
    with coordinator.lock:
        coordinator.active -= ids
        coordinator.index = {key: value for key, value in coordinator.index.items() if key[0] not in ids}
        coordinator.sources = {key: value for key, value in coordinator.sources.items() if key[0] not in ids}
        for ied_id in ids:
            coordinator.ied_indexes.pop(ied_id, None)
            coordinator.sample_writer.locks.pop(ied_id, None)
    coordinator.mms_access.forget(ids)
