"""Apply values received from a real IED to the authoritative signal index."""
from studio_app.domain.live_value import convert

SOURCES = {"live_report", "live_poll", "live_goose"}
COMPANIONS = {"Quality", "Timestamp"}


class LiveValues:
    def __init__(self, coordinator):
        self.coordinator = coordinator

    def apply(self, ied_id, raw_values, source):
        """raw_values: {ref: device JSON}. Unknown refs are ignored, bad values reported."""
        if source not in SOURCES:
            raise ValueError("Unknown live source")
        coordinator = self.coordinator
        index = coordinator.ied_indexes.get(ied_id, {})
        changed, errors = {}, {}
        for reference, raw in raw_values.items():
            signal = index.get(reference)
            if signal is None:
                continue
            try:
                changed[reference] = convert(signal["kind"], raw)
            except ValueError as error:
                errors[reference] = str(error)
        with coordinator.lock:
            for reference, value in changed.items():
                index[reference]["value"] = value
                index[reference]["live_state"] = "fresh"
        emitted = set()
        for reference, value in changed.items():
            signal = index[reference]
            owner = self._owner(index, signal) if signal["kind"] in COMPANIONS else None
            target = owner or signal
            if target["ref"] in emitted:
                continue
            emitted.add(target["ref"])
            refs = [target["ref"], target.get("quality_ref"), target.get("timestamp_ref")]
            updates = {ref: index[ref]["value"] for ref in refs if ref and ref in changed}
            quality = index[target["quality_ref"]]["value"] if target.get("quality_ref") in index else 0
            stamp = index[target["timestamp_ref"]]["value"] if target.get("timestamp_ref") in index else 0
            coordinator.emit("signal", ied_id=ied_id, ref=target["ref"], value=target["value"], source=source,
                             quality=quality, source_timestamp=stamp, updates=updates, live_state="fresh")
        for reference, message in errors.items():
            coordinator.emit("live_read_error", ied_id=ied_id, ref=reference, message=message)
        coordinator.equipment_states.publish_changed((ied_id, reference) for reference in changed)
        return {"applied": len(changed), "errors": errors}

    @staticmethod
    def _owner(index, companion):
        base = companion["ref"].rsplit(".", 1)[0]
        for name in ("stVal", "mag.f", "mag.i", "general", "cVal.mag.f", "setVal"):
            candidate = index.get(f"{base}.{name}")
            if candidate and (candidate.get("quality_ref") == companion["ref"]
                              or candidate.get("timestamp_ref") == companion["ref"]):
                return candidate
        return None

    def mark_stale(self, ied_id, reason):
        """Connection lost: keep the last values visible but never present them as fresh."""
        coordinator = self.coordinator
        with coordinator.lock:
            for signal in coordinator.ied_indexes.get(ied_id, {}).values():
                if signal.get("live_state") == "fresh":
                    signal["live_state"] = "stale"
        coordinator.emit("live_stale", ied_id=ied_id, message=reason)
