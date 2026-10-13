"""Validate portable numeric source definitions without runtime dependencies."""
import math
from .csv_trace import validate_trace


def validate_source(signal, spec):
    if signal["fc"] != "MX" or not signal["kind"].startswith(("FLOAT", "INT")):
        raise ValueError("Automatic sources require numeric MX measurements")
    if not isinstance(spec, dict) or spec.get("mode") not in {
            "constant", "ramp", "sine", "step", "noise", "manual", "csv"}:
        raise ValueError("Unsupported source mode")
    if spec["mode"] == "csv":
        return validate_trace(signal, spec)
    candidate = {"mode": spec["mode"], "offset": float(spec.get("offset", 0)),
                 "amplitude": float(spec.get("amplitude", 1)),
                 "period": float(spec.get("period", 10)), "seed": int(spec.get("seed", 1)),
                 "overridden": False}
    if not all(math.isfinite(candidate[key]) for key in ("offset", "amplitude", "period")):
        raise ValueError("Source parameters must be finite numbers")
    if candidate["period"] < 0.1:
        raise ValueError("Source period must be at least 0.1 seconds")
    return candidate


def source_summary(spec):
    return {key: value for key, value in spec.items() if key not in {"csv_text", "points"}}
