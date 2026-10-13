"""Find the closest same-FC quality/time companions of an SCL leaf attribute."""
from .signal import validate_value


def companions(signal, by_reference):
    result = {}
    parent = signal["ref"].rsplit(".", 1)[0]
    while "." in parent:
        for suffix, kind in (("q", "Quality"), ("t", "Timestamp")):
            candidate = by_reference.get(parent + "." + suffix)
            if (suffix not in result and candidate and candidate["kind"] == kind
                    and candidate["fc"] == signal["fc"]):
                result[suffix] = candidate
        parent = parent.rsplit(".", 1)[0]
    return result


def make_sample(signal, by_reference, value, timestamp, quality=None):
    updates = [{"ref": signal["ref"], "value": validate_value(signal, value)}]
    related = companions(signal, by_reference)
    if signal["kind"] in {"Quality", "Timestamp"}:
        return updates, related
    if quality is not None:
        if "q" not in related:
            raise ValueError("This attribute has no associated SCL quality")
        updates.append({"ref": related["q"]["ref"], "value": validate_value(related["q"], quality)})
    if "t" in related:
        updates.append({"ref": related["t"]["ref"], "value": validate_value(related["t"], timestamp)})
    return updates, related
