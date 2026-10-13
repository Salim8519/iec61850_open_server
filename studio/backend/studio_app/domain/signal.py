"""Validate engineering scalar values independently of transport and persistence."""
import math


def validate_value(signal, value):
    kind = signal["kind"]
    if kind == "BOOLEAN":
        if not isinstance(value, bool):
            raise ValueError("Boolean signals require true or false")
        return value
    if kind.startswith("FLOAT"):
        if isinstance(value, bool) or not isinstance(value, (int, float)) or not math.isfinite(value):
            raise ValueError("Measurement must be a finite number")
        if kind == "FLOAT32" and abs(value) > 3.4028235e38:
            raise ValueError("Measurement exceeds FLOAT32 range")
        return float(value)
    if kind.startswith("INT") or kind in {"Enum", "Dbpos", "Tcmd", "Quality", "Timestamp"}:
        if isinstance(value, bool) or not isinstance(value, int):
            raise ValueError("This signal requires an integer")
        if kind.startswith("INT"):
            unsigned = kind.endswith("U")
            width = int(kind[3:].rstrip("U"))
            minimum = 0 if unsigned else -(1 << (width - 1))
            maximum = (1 << (width if unsigned else width - 1)) - 1
        elif kind in {"Dbpos", "Tcmd"}:
            minimum, maximum = 0, 3
        elif kind == "Quality":
            minimum, maximum = 0, 8191
        elif kind == "Timestamp":
            minimum, maximum = 0, 4294967295999
        else:
            minimum, maximum = -(1 << 31), (1 << 63) - 1
        if not minimum <= value <= maximum:
            raise ValueError(f"Value must be between {minimum} and {maximum}")
        return value
    if "String" in kind or kind in {"ObjRef", "Currency"}:
        size = int(kind.split("String")[-1]) if "String" in kind else 129
        if not isinstance(value, str) or len(value.encode("utf-8")) > size:
            raise ValueError(f"Text must be at most {size} UTF-8 bytes")
        return value
    raise ValueError(f"Manual editing of {kind} is not implemented")
