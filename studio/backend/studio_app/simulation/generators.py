"""Deterministic engineering-value generators."""
import math
import random
from studio_app.domain.csv_trace import trace_value


def value_at(spec, elapsed):
    elapsed = max(0, elapsed - spec.get("start_elapsed", 0))
    kind = spec.get("mode", "constant")
    if kind == "csv":
        return trace_value(spec, elapsed)
    offset = float(spec.get("offset", 0))
    amplitude = float(spec.get("amplitude", 1))
    period = max(0.1, float(spec.get("period", 10)))
    if kind == "constant":
        return offset
    if kind == "ramp":
        return offset + amplitude * ((elapsed % period) / period)
    if kind == "sine":
        return offset + amplitude * math.sin(2 * math.pi * elapsed / period)
    if kind == "step":
        return offset + (amplitude if (elapsed % period) >= period / 2 else 0)
    if kind == "noise":
        return offset + random.Random(int(spec.get("seed", 1)) + int(elapsed * 10)).uniform(-amplitude, amplitude)
    raise ValueError(f"Unknown generator {kind}")
