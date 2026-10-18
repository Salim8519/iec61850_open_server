"""Apply equipment test buttons to simulated IEDs through the sample writer (ADR-0067)."""
from studio_app.contracts.errors import StudioError
from studio_app.domain.equipment_test import button_writes

REAL_ONLY = ("{name} is a real device. Test buttons only change IEDs that ScadaBench simulates; "
             "operate real equipment with its command link.")


def apply_equipment_test(coordinator, node_id, action, value=None, row=None):
    """Write the bound point(s) as if the equipment changed by itself; no command is sent."""
    node = next((item for item in coordinator.projects.project["diagram"]["nodes"] if item["id"] == node_id), None)
    if node is None:
        raise StudioError("Unknown equipment")
    if coordinator.state not in {"running", "paused"}:
        raise StudioError("Run the IED before using a test button")
    kind_of = lambda binding: (coordinator.index.get((binding["ied_id"], binding["ref"])) or {}).get("kind")
    try:
        writes = button_writes(node["data"], action, value, row, kind_of)
    except ValueError as error:
        raise StudioError(str(error)) from error
    grouped = {}
    for binding, item, quality in writes:
        ied = coordinator.projects.find_ied(binding["ied_id"])
        if ied.get("role", "simulated") != "simulated":
            raise StudioError(REAL_ONLY.format(name=ied["name"]))
        if ied["id"] not in coordinator.sample_writer.locks:
            raise StudioError(f"{ied['name']} is not running. Run this IED first.")
        signal = coordinator.index[(ied["id"], binding["ref"])]
        if item is None:  # quality only: keep the present value
            item = signal["value"]
        if quality is not None and not signal.get("quality_ref"):
            if quality:
                raise StudioError("This point has no quality attribute in its SCL, so its quality cannot be set")
            quality = None
        grouped.setdefault(ied["id"], []).append((binding["ref"], item, quality))
    written = {}
    with coordinator.cycle_lock:
        for ied_id, items in grouped.items():
            written[ied_id] = coordinator.sample_writer.apply(ied_id, items, "equipment_test")
    coordinator.emit("equipment_test", node_id=node_id, action=action)
    return {"node_id": node_id, "action": action, "written": written}
