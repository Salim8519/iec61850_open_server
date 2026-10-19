"""Choose deterministic value overrides and record export_ied results for one IED."""
import hashlib
import pathsetup
from golden_common import capture, write_json
from studio_app.scl.exporter import export_ied

KINDS = ["BOOLEAN", "INT", "FLOAT", "VisString"]


def _family(kind):
    return next((family for family in KINDS if kind.startswith(family)), None)


def _int_range(kind):
    unsigned = kind.endswith("U")
    width = int(kind[3:].rstrip("U"))
    low = 0 if unsigned else -(1 << (width - 1))
    return low, (1 << (width if unsigned else width - 1)) - 1


def _new_value(signal):
    family = _family(signal["kind"])
    if family == "BOOLEAN":
        return not signal["value"]
    if family == "INT":
        low, high = _int_range(signal["kind"])
        return signal["value"] + 1 if signal["value"] + 1 <= high else max(low, signal["value"] - 1)
    if family == "FLOAT":
        return 12.5
    return "x"


def _pick(case, items):
    return items[int(hashlib.sha256(case.encode("utf-8")).hexdigest()[:8], 16) % len(items)]


def candidates(signals):
    """Writable scalar signals by family; per-index references are never exported."""
    result = {family: [] for family in KINDS}
    for signal in signals:
        family = _family(signal["kind"])
        if family and signal["writable"] and "(" not in signal["ref"] and signal["kind"] != "INT128":
            result[family].append(signal)
    return result


def choose_overrides(case, index, signals):
    """1-3 overrides; which families are used rotates with the case name and IED index."""
    pool = candidates(signals)
    start = (int(hashlib.sha256(case.encode("utf-8")).hexdigest()[8:16], 16) + index) % len(KINDS)
    order = KINDS[start:] + KINDS[:start]
    chosen = [_pick(f"{case}/{family}", pool[family]) for family in order if pool[family]][:3]
    return {signal["ref"]: _new_value(signal) for signal in chosen}


def _wrong_type(signal):
    return {"BOOLEAN": 1, "INT": "1", "FLOAT": "x", "VisString": 5}[_family(signal["kind"])]


def bad_overrides(signals):
    """A few override sets that the exporter must refuse, with their exact messages."""
    pool = candidates(signals)
    first = next((pool[family][0] for family in KINDS if pool[family]), None)
    result = [{"Nope/LLN0.Beh.stVal": 1}, ["not", "a", "dict"], "text"]
    if first is not None:
        result.append({first["ref"]: _wrong_type(first)})
    for family, value in (("INT", 1 << 70), ("FLOAT", 1e39), ("VisString", "y" * 300), ("BOOLEAN", None)):
        if pool[family]:
            result.append({pool[family][0]["ref"]: value})
    for kind, value in (("Enum", 99), ("Dbpos", 2), ("Timestamp", 1700000000123)):
        signal = next((item for item in signals if item["kind"] == kind and item["writable"]
                       and "(" not in item["ref"]), None)
        if signal is not None:
            result.append({signal["ref"]: value})
    return result


def export_one(folder, case, index, text, name, signals, files):
    overrides = choose_overrides(case, index, signals)
    if not overrides:
        return False
    write_json(folder, f"export-{index}.overrides.json", overrides, files)
    ok, payload = capture(export_ied, text, name, overrides)
    if ok:
        pathsetup.write_bytes(folder / f"export-{index}.scd", payload)
        files.append(f"export-{index}.scd")
    else:
        write_json(folder, f"export-{index}.error.json", payload, files)
    return True


def export_errors(folder, text, name, signals, files):
    results = []
    for number, overrides in enumerate(bad_overrides(signals)):
        ok, payload = capture(export_ied, text, name, overrides)
        entry = {"ied": name, "overrides": overrides}
        if ok:
            entry["payload"] = f"export-errors-{number}.scd"
            pathsetup.write_bytes(folder / entry["payload"], payload)
            files.append(entry["payload"])
        else:
            entry["error"] = payload
        results.append(entry)
    write_json(folder, "export-errors.json", results, files)
