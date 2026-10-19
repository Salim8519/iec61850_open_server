"""Random master_values batches (what the native master worker reports) for the oracles."""
import json

from studio_app.domain import master_points as mp

GROUP_NAMES = list(mp.GROUPS)
QUALITIES = [0, 0, 0, 1, 2, 0x80, 0x40, 0x20, 0x10, 0x01, 0x21, 0x41, 0x61, 0xff, 0x7e, -1, 2 ** 70 + 0x21, True]


def master_value(r, odd=0.04):
    """One master_values entry, sometimes malformed."""
    if r.random() < odd / 2:
        return r.choice([None, 5, "x", [1]])
    value = {"group": r.choice(GROUP_NAMES * 3 + ["bogus", None, 5]),
             "address": r.choice([r.randint(0, 20)] * 6 + [r.randint(0, 70000), 16777215, 0, 1, True, False,
                                                           2 ** 70, -3, 1.0, "1", None])}
    if r.random() < 0.9:
        value["value"] = r.choice([True, False, 0, 1, 2, 3, 4, 2.0, 1.5, -0.0, 1e-7, 123456789.0, float("nan"),
                                   float("inf"), "on", None, 2 ** 70, -7])
        if r.random() < odd:
            value["value"] = r.choice([[1], {"a": 1}])
    if r.random() < 0.9:
        value["quality"] = r.choice(QUALITIES)
        if r.random() < odd:
            value["quality"] = r.choice([1.9, "3", "x", None, [1], float("nan"), float("inf"), " 7 "])
    for key, values in (("type", ["uint16", "int16", "uint32", "int32", "float32", "", None]),
                        ("time_ms", [0, None, 1700000000000, 5.5]), ("cause", ["spontaneous", "", 3])):
        if r.random() < 0.7:
            value[key] = r.choice(values)
    if r.random() < odd / 2:
        value["type"] = ["x"]
    return value


def merge_batches(r, odd=0.04):
    return [[master_value(r, odd) for _ in range(r.randint(0, 25))] for _ in range(r.randint(1, 4))]


def merge_sequence(batches):
    """Apply the batches to one table: per batch the new keys (or the error) and the table."""
    table, steps = {}, []
    for batch in batches:
        try:
            steps.append({"added": mp.merge(table, json.loads(json.dumps(batch)))})
        except Exception as error:  # the table keeps what Python changed before the error
            steps.append({"error": f"{type(error).__name__}: {error}"})
        steps.append({"table": json.loads(json.dumps(table))})
    return steps, table
