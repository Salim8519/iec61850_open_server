"""Batch acknowledged samples per IED; serialize manual and automatic writes."""
import threading
import time
from studio_app.domain.sample import make_sample


class SampleWriter:
    def __init__(self, coordinator):
        self.coordinator = coordinator
        self.locks = {}

    def reset(self, ieds):
        self.locks = {ied["id"]: threading.RLock() for ied in ieds}

    def apply(self, ied_id, items, source, guard=None):
        coordinator = self.coordinator
        if ied_id not in self.locks:
            raise ValueError("IED is not running")
        with self.locks[ied_id]:
            if coordinator.state not in {"running", "paused"}:
                raise ValueError("Run the IED before updating a live value")
            index = coordinator.ied_indexes[ied_id]
            updates, samples = {}, []
            timestamp = int(time.time() * 1000)
            for reference, value, quality in items:
                signal = index.get(reference)
                if signal is None or not signal["writable"] or signal["fc"] == "CF":
                    raise ValueError("This attribute is not runtime-editable")
                spec = coordinator.sources.get((ied_id, reference))
                if source == "generator" and (not spec or spec.get("overridden")):
                    continue
                batch, related = make_sample(signal, index, value, timestamp, quality)
                updates.update({item["ref"]: item["value"] for item in batch})
                samples.append((signal, related))
            if not updates:
                return {}
            if len(updates) > 256:
                raise ValueError("Sample batch exceeds the native 256-attribute limit")
            batch = [{"ref": reference, "value": value} for reference, value in updates.items()]
            guarded = coordinator.supervisor.update_guarded(ied_id, batch, guard) if guard else None
            result = guarded["values"] if guard else coordinator.supervisor.update_sample(ied_id, batch)
            with coordinator.lock:
                for reference, actual in result.items():
                    index[reference]["value"] = actual
                for signal, related in samples:
                    signal.update(timestamp=related.get("t", {}).get("value", 0),
                                  quality=related.get("q", {}).get("value", 0))
                    if source in {"manual", "scenario", "pulse", "pulse_return",
                                  "control_feedback", "equipment_button", "equipment_test"} and (ied_id, signal["ref"]) in coordinator.sources:
                        coordinator.sources[(ied_id, signal["ref"])]["overridden"] = True
            for signal, related in samples:
                if source not in {"pulse", "pulse_return"} and hasattr(coordinator, "pulses"):
                    coordinator.pulses.cancel(ied_id, signal["ref"], "Newer signal write")
                refs = [signal["ref"]] + [item["ref"] for item in related.values()]
                coordinator.emit("signal", ied_id=ied_id, ref=signal["ref"], value=result[signal["ref"]],
                                 source=source, quality=signal["quality"], source_timestamp=signal["timestamp"],
                                 updates={reference: result[reference] for reference in refs if reference in result})
            coordinator.equipment_states.publish_changed((ied_id, reference) for reference in result)
            return guarded if guard else result
