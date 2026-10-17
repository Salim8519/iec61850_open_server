"""The value a gateway point carries right now, from the authoritative signal index."""


def ied_online(ied, run_state):
    """Simulated IEDs need Run; real IEDs need a live connection."""
    if run_state not in ("running", "paused"):
        return False
    if ied.get("role") == "real":
        return ied.get("state") == "real_live"
    return ied.get("state") == "running"


def sample(point, index, ied, run_state):
    """{"id", "value", "quality", "time_ms", "source_ok"} for one monitoring point.

    index maps (ied_id, ref) to signal records with value, quality_ref and timestamp_ref.
    A real IED value counts only while it is fresh; otherwise the point is invalid.
    """
    signal = index.get((point["ied"], point["ref"]))
    if signal is None or ied is None:
        return {"id": point["id"], "value": 0, "quality": 0, "time_ms": 0, "source_ok": False}
    value = signal.get("value")
    if isinstance(value, bool):
        value = int(value)
    if not isinstance(value, (int, float)):
        value = 0
    quality_signal = index.get((point["ied"], signal.get("quality_ref"))) if signal.get("quality_ref") else None
    time_signal = index.get((point["ied"], signal.get("timestamp_ref"))) if signal.get("timestamp_ref") else None
    quality = quality_signal.get("value", 0) if quality_signal else 0
    online = ied_online(ied, run_state) and signal.get("live_state", "fresh") == "fresh"
    return {"id": point["id"], "value": value, "quality": quality if isinstance(quality, int) else 0,
            "time_ms": time_signal.get("value", 0) if time_signal else 0, "source_ok": online}


def display(point, current):
    """Plain text of a point value for the point table."""
    if not current["source_ok"]:
        return "no value (IED not running or not connected)"
    value = current["value"]
    if point["kind"] == "single":
        text = "ON" if value else "OFF"
    elif point["kind"] == "double":
        text = {0: "intermediate", 1: "OFF (open)", 2: "ON (closed)", 3: "bad state"}.get(value, str(value))
    else:
        text = f"{value * point.get('scale', 1.0):.6g}"
    validity = current["quality"] & 3
    return text + (" · invalid" if validity == 2 else " · questionable" if validity == 3 else "")
