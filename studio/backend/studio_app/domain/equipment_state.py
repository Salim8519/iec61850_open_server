"""Pure construction of equipment display states from exact signal bindings."""


def binding_keys(mapping):
    if not mapping:
        return []
    if mapping["mode"] == "contacts_to_dbpos":
        bindings = (mapping["open"], mapping["closed"])
    else:
        bindings = (mapping["source"],)
    return [(item["ied_id"], item["ref"]) for item in bindings]


def evaluate_state(mapping, values):
    """Return a raw direct value or normalized IEC 61850 Dbpos integer."""
    if mapping["mode"] == "direct":
        return values(binding_keys(mapping)[0])
    if mapping["mode"] == "single_to_dbpos":
        value = values(binding_keys(mapping)[0])
        return 2 if value is True else 1 if value is False else None
    open_value = values((mapping["open"]["ied_id"], mapping["open"]["ref"]))
    closed_value = values((mapping["closed"]["ied_id"], mapping["closed"]["ref"]))
    if not isinstance(open_value, bool) or not isinstance(closed_value, bool):
        return None
    if open_value is True and closed_value is False:
        return 1
    if open_value is False and closed_value is True:
        return 2
    return 3 if open_value is True and closed_value is True else 0
