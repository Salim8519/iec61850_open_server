"""JSON parsing that keeps number lexemes, so `1` vs `1.0` is visible to the diff."""
import json


class Num(str):
    """A JSON number kept as its exact text (`1`, `1.0`, `1e-05`, `NaN`)."""

    @property
    def is_float(self):
        return any(mark in self for mark in ".eEN") or self in ("Infinity", "-Infinity")

    def value(self):
        return float(self) if self.is_float else int(self)


def loads(text):
    return json.loads(text, parse_float=Num, parse_int=Num, parse_constant=Num)


def has_nan(tree):
    if isinstance(tree, Num):
        return tree in ("NaN", "Infinity", "-Infinity")
    if isinstance(tree, dict):
        return any(has_nan(value) for value in tree.values())
    if isinstance(tree, list):
        return any(has_nan(value) for value in tree)
    return False
