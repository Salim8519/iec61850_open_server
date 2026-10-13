"""Pure type-driven equipment command and simulated feedback rules."""

SWITCHES = {"CB", "DS", "ES"}


def expected_cdc(kind):
    if kind in SWITCHES:
        return "DPC"
    if kind in {"TRIP_RELAY", "PUSH_BUTTON"}:
        return "SPC"
    raise ValueError("This equipment type is not controllable")


def semantic_command(data, action=None, value=None):
    kind = data["kind"]
    if action is not None and value is not None:
        raise ValueError("Send an equipment action or a raw value, not both")
    if action is None:
        if not isinstance(value, bool):
            raise ValueError("Control value must be boolean")
        if kind in SWITCHES:
            return ("close" if value else "open"), value
        if kind == "TRIP_RELAY":
            operated = data.get("trip_command_polarity", "operate_false") == "operate_true"
            return ("operate" if value == operated else "reset"), value
        if kind == "PUSH_BUTTON":
            return "press", value
        raise ValueError("This equipment type is not controllable")
    if kind in SWITCHES and action in {"open", "close"}:
        return action, action == "close"
    if kind == "TRIP_RELAY" and action in {"operate", "reset"}:
        operated = data.get("trip_command_polarity", "operate_false") == "operate_true"
        return action, operated if action == "operate" else not operated
    if (kind == "PUSH_BUTTON" and data.get("push_mode") == "spc_momentary"
            and action == "press"):
        return action, bool(data.get("push_value", True))
    if (kind == "PUSH_BUTTON" and data.get("push_mode") == "self_resetting"
            and action == "press"):
        return action, not bool(data.get("push_rest_value", False))
    raise ValueError("Action is incompatible with this equipment")


def feedback_targets(data, mapping, action):
    if not mapping:
        return []
    kind = data["kind"]
    mode = mapping["mode"]
    if kind == "TRIP_RELAY":
        if mode != "direct":
            raise ValueError("Trip relay feedback requires one Boolean state point")
        if action not in {"operate", "reset"}:
            raise ValueError("Trip relay feedback requires an Operate or Reset action")
        operated = data.get("trip_state_polarity", "operated_true") == "operated_true"
        return [(mapping["source"], operated if action == "operate" else not operated)]
    if kind not in SWITCHES:
        raise ValueError("This equipment type is not controllable")
    if action not in {"open", "close"}:
        raise ValueError("Switch feedback requires an Open or Close action")
    value = action == "close"
    if mode == "direct":
        return [(mapping["source"], 2 if value else 1)]
    if mode == "single_to_dbpos":
        return [(mapping["source"], value)]
    if mode == "contacts_to_dbpos":
        return [(mapping["open"], not value), (mapping["closed"], value)]
    raise ValueError("Unsupported equipment state mapping")
