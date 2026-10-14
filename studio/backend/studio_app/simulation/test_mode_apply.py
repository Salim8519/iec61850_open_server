"""Apply saved test modes to Studio's simulated IEDs (never to real devices)."""
from studio_app.contracts.errors import StudioError
from studio_app.domain.test_mode import mode_writes, validate_test_mode


def _write(coordinator, ied):
    if ied.get("role") != "simulated" or ied["id"] not in coordinator.sample_writer.locks:
        return 0
    index = coordinator.ied_indexes.get(ied["id"], {})
    current = {ref: signal["value"] for ref, signal in index.items() if ref.endswith(".Mod.stVal")}
    writes = mode_writes(ied, ied.get("test_mode") or validate_test_mode(None), current)
    if writes:
        coordinator.sample_writer.apply(ied["id"], [(ref, value, None) for ref, value in writes], "test_mode")
    return len(writes)


def apply_test_modes(coordinator):
    """At Run start: write every simulated IED's saved modes that differ from normal service."""
    for ied in coordinator.projects.project["ieds"]:
        settings = ied.get("test_mode") or {}
        if settings.get("devices") or settings.get("simulation"):
            _write(coordinator, ied)


def set_test_mode(coordinator, ied_id, value):
    """Save the modes of one simulated IED and apply them at once when it is running."""
    projects = coordinator.projects
    with projects.lock:
        ied = projects.find_ied(ied_id)
        if ied.get("role") != "simulated":
            raise StudioError("Mode and simulation can only be set on IEDs that Studio simulates. "
                              "Studio never changes a real device.")
        try:
            settings = validate_test_mode(value)
        except ValueError as error:
            raise StudioError(f"{ied['name']}: {error}") from error
        ied["test_mode"] = settings
        projects.save()
        projects.changed()
    written = _write(coordinator, ied) if coordinator.state in {"running", "paused"} else 0
    return {"test_mode": settings, "written": written}
