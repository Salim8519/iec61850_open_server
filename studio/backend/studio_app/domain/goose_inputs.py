"""Bind a simulated subscriber's GOOSE inputs (ExtRef intAddr) to its own data points. Pure.

An input is applied only when its intAddr names exactly one runtime-editable data
attribute of the subscriber: the full reference, the reference inside the input's own
logical device, or a unique `/LN.DO.DA` match. Anything else is a worded diagnostic;
ScadaBench never guesses what a vendor-specific intAddr means (ADR-0066).
"""
from .live_value import convert


def _target(int_addr, ld, signals):
    if not int_addr or " " in int_addr or "|" in int_addr:
        return None, "its input address is not a data point of this IED"
    for candidate in (int_addr, f"{ld}/{int_addr}" if ld else None):
        if candidate and candidate in signals:
            return candidate, ""
    matches = [ref for ref in signals if ref.endswith("/" + int_addr)]
    if len(matches) == 1:
        return matches[0], ""
    if len(matches) > 1:
        return None, "its input address matches more than one data point"
    return None, "its input address is not a data point of this IED"


def bind_inputs(ied, subscription):
    """(bindings, problems) for one subscription of one simulated subscriber IED."""
    signals = {signal["ref"]: signal for signal in ied.get("signals", [])}
    domain = (subscription.get("dataset_ref") or "").split("/", 1)[0]
    bindings, problems = [], []
    for item in subscription.get("inputs", []):
        label = item.get("desc") or item.get("int_addr") or item.get("source", "")
        if item.get("member") is None:
            problems.append(f"Input {label} is not in the dataset of {subscription['gocb_ref']}")
            continue
        target, reason = _target(item.get("int_addr", ""), item.get("ld", ""), signals)
        if target is None:
            problems.append(f"Input {label} is shown but not applied: {reason}")
            continue
        signal = signals[target]
        if not signal.get("writable") or signal.get("fc") == "CF":
            problems.append(f"Input {label} is shown but not applied: {target} cannot be changed while running")
            continue
        source = item.get("source", "").split("/", 2)
        bindings.append({"target": target, "kind": signal["kind"], "member": item["member"],
                         "path": item.get("member_path", ""), "desc": label,
                         "source_ref": f"{domain}/{source[2]}" if domain and len(source) == 3 else ""})
    return bindings, problems


def accept_frame(event, member_count):
    """Reason the received frame must not drive inputs, or '' when it may."""
    if event.get("state") != "receiving" or event.get("problem"):
        return f"GOOSE is {event.get('problem') or event.get('state')}"
    values = event.get("values")
    if not isinstance(values, list):
        return "the frame carried no dataset values"
    if member_count and len(values) != member_count:
        return "the dataset size differs from the SCD"
    return ""


def input_updates(bindings, values, publisher_values=None):
    """[(target, value)] for every binding whose value can be read without guessing, plus problems."""
    updates, problems = [], []
    for binding in bindings:
        raw = values[binding["member"]] if binding["member"] < len(values) else None
        if isinstance(raw, list):
            raw = (publisher_values or {}).get(binding["source_ref"]) if binding["path"] else None
            if raw is None:
                problems.append(f"Input {binding['desc']}: the member is a structure; its layout is known "
                                "only when the publisher's file is loaded")
                continue
        try:
            updates.append((binding["target"], convert(binding["kind"], raw)))
        except ValueError as error:
            problems.append(f"Input {binding['desc']}: {error}")
    return updates, problems
