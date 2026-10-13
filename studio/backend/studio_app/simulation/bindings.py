"""Populate mapped canvas values from the native model when a run starts."""
from studio_app.domain.equipment_state import binding_keys


def publish_bindings(coordinator):
    bindings = {key for node in coordinator.projects.project["diagram"]["nodes"]
                for key in binding_keys(node["data"].get("state_mapping"))}
    bindings.update((entry["ied_id"], entry["ref"]) for entry in coordinator.projects.project.get("test_panel", []))
    for ied_id, reference in bindings:
        if not ied_id or not reference or ied_id not in coordinator.supervisor.workers:
            continue
        signal = coordinator.index.get((ied_id, reference))
        if not signal:
            continue
        refs = [reference] + [signal[key] for key in ("quality_ref", "timestamp_ref") if signal.get(key)]
        values = {ref: coordinator.supervisor.read(ied_id, ref) for ref in refs}
        with coordinator.lock:
            for ref, value in values.items():
                coordinator.index[(ied_id, ref)]["value"] = value
        coordinator.emit("signal", ied_id=ied_id, ref=reference, value=values[reference],
                         quality=values.get(signal.get("quality_ref"), 0),
                         source_timestamp=values.get(signal.get("timestamp_ref"), 0),
                         source="initial_native_snapshot", updates=values)
    coordinator.equipment_states.publish_all()
