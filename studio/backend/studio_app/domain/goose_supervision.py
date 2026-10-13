"""Pure GOOSE bookkeeping: what to watch, LGOS supervision and dataset values."""

LGOS_POINTS = {"St.stVal": "receiving", "SimSt.stVal": "test", "LastStNum.stVal": "st_num",
               "ConfRevNum.stVal": "conf_rev"}


def watch_list(ieds):
    """Every GoCB to observe, keyed by reference, with who publishes and subscribes."""
    watches = {}
    for ied in ieds:
        role = ied.get("role", "simulated")
        if role == "off":
            continue
        goose = ied.get("goose") or {}
        for control in goose.get("publishes", []):
            entry = watches.setdefault(control["gocb_ref"], _entry(control))
            entry["publisher"] = {"ied_id": ied["id"], "name": ied["name"], "role": role}
        if role != "simulated":
            continue
        for subscription in goose.get("subscribes", []):
            entry = watches.setdefault(subscription["gocb_ref"], _entry(subscription))
            entry.setdefault("publisher", {"ied_id": None, "name": subscription.get("publisher", ""),
                                           "role": "outside_project"})
            entry["subscribers"].append({"ied_id": ied["id"], "name": ied["name"],
                                         "lgos": subscription.get("lgos"),
                                         "inputs": len(subscription.get("inputs", []))})
    return watches


def _entry(control):
    return {"key": control["gocb_ref"], "gocb_ref": control["gocb_ref"], "app_id": control["app_id"],
            "mac": control["mac"], "go_id": control["go_id"], "dataset_ref": control["dataset_ref"],
            "conf_rev": control["conf_rev"], "subscribers": []}


def lgos_updates(lgos_ln, received, index):
    """LGOS attribute writes for one subscriber; only LGOS points that exist are touched."""
    if not lgos_ln:
        return []
    values = {"receiving": received.get("state") == "receiving", "test": bool(received.get("test")),
              "st_num": received.get("st_num"), "conf_rev": received.get("conf_rev")}
    updates = []
    for suffix, field in LGOS_POINTS.items():
        reference = f"{lgos_ln}.{suffix}"
        signal = index.get(reference)
        value = values[field]
        if signal is None or value is None:
            continue
        if signal["kind"] == "BOOLEAN":
            value = bool(value)
        elif not isinstance(value, int) or isinstance(value, bool):
            continue
        updates.append((reference, value))
    return updates


def _leaves(value):
    if isinstance(value, list):
        for item in value:
            yield from _leaves(item)
    else:
        yield value


def dataset_values(ied, control, values):
    """Map received GOOSE dataset values to the publisher's SCD signal references."""
    dataset = next((item for item in ied.get("datasets", []) if item.get("ref") == control["dataset_ref"]), None)
    if dataset is None or not isinstance(values, list) or len(values) != len(dataset.get("fcda_refs", [])):
        return {}
    signals = ied.get("signals", [])
    result = {}
    for member, fcda_ref, value in zip(control["members"], dataset["fcda_refs"], values):
        parts = member.split("/", 1)[-1].split("$")
        fc = parts[1] if len(parts) > 1 else "ST"
        if not isinstance(value, list):
            result[fcda_ref] = value
            continue
        targets = [s["ref"] for s in signals if s["fc"] == fc and
                   (s["ref"].startswith(fcda_ref + ".") or s["ref"].startswith(fcda_ref + "("))]
        leaves = list(_leaves(value))
        if len(targets) == len(leaves):
            result.update(zip(targets, leaves))
    return result
