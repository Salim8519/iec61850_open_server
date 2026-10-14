"""Check cross-references before replacing an engineering project."""
from studio_app.contracts.errors import StudioError
from studio_app.contracts.project import Diagram
from studio_app.domain.source import validate_source
from studio_app.domain.signal import validate_value
from .test_panel import validate_mapping

SWITCHES = {"CB", "DS", "ES"}
DRAWING_ONLY = {"JUNCTION": "A junction", "INFEED": "An infeed"}
INFEED_FIELDS = ("infeed_type", "infeed_state")
INFEED_DEFAULTS = {"infeed_type": "line", "infeed_state": "live"}
CONTROLLABLE = SWITCHES | {"TRIP_RELAY", "PUSH_BUTTON"}
NUMERIC = {"INT8", "INT16", "INT32", "INT64", "INT128", "INT8U", "INT16U",
           "INT24U", "INT32U", "FLOAT32", "FLOAT64"}


def _binding_signal(binding, ieds):
    ied = next((item for item in ieds if item["id"] == binding["ied_id"]), None)
    if ied is None:
        raise StudioError("A signal binding references an unknown IED")
    signal = next((item for item in ied["signals"] if item["ref"] == binding["ref"]), None)
    if signal is None:
        raise StudioError("A signal binding references an unknown signal")
    if binding.get("report_ref"):
        validate_mapping(ied, binding["ref"], binding["report_ref"])
    return signal


def _validate_signal_binding(binding, ieds, expected=None):
    signal = _binding_signal(binding, ieds)
    if expected and signal["kind"] != expected:
        raise StudioError(f"State mapping requires a {expected} signal")
    if not signal["writable"] or signal["fc"] == "CF":
        raise StudioError("State mapping requires a runtime-editable status signal")
    return signal


def _validate_state(data, ieds):
    mapping = data.get("state_mapping")
    if not mapping:
        return
    kind, mode = data["kind"], mapping["mode"]
    if kind in DRAWING_ONLY:
        raise StudioError(f"{DRAWING_ONLY[kind]} is a drawing point and does not accept a state mapping")
    if kind == "RELAY":
        raise StudioError("A relay uses named status entries instead of a single state mapping")
    if kind in SWITCHES:
        if mode == "direct":
            _validate_signal_binding(mapping["source"], ieds, "Dbpos")
        elif mode == "single_to_dbpos":
            _validate_signal_binding(mapping["source"], ieds, "BOOLEAN")
        else:
            opened = mapping["open"]
            closed = mapping["closed"]
            if (opened["ied_id"], opened["ref"]) == (closed["ied_id"], closed["ref"]):
                raise StudioError("Open and closed contacts must be different signals")
            _validate_signal_binding(opened, ieds, "BOOLEAN")
            _validate_signal_binding(closed, ieds, "BOOLEAN")
    elif kind == "TRIP_RELAY":
        if mode != "direct":
            raise StudioError("Trip relay state requires one Boolean signal")
        _validate_signal_binding(mapping["source"], ieds, "BOOLEAN")
    elif kind == "PUSH_BUTTON":
        if data.get("push_mode") not in {"latching_status", "self_resetting"} or mode != "direct":
            raise StudioError("Only a latching or self-resetting push button accepts one Boolean state signal")
        _validate_signal_binding(mapping["source"], ieds, "BOOLEAN")
    elif kind == "SOURCE":
        if mode != "direct":
            raise StudioError("Analog source requires one numeric MX signal")
        signal = _validate_signal_binding(mapping["source"], ieds)
        if signal["fc"] != "MX" or signal["kind"] not in NUMERIC:
            raise StudioError("Analog source requires one numeric MX signal")
    elif mode != "direct":
        raise StudioError("Constructed double-point state is only valid for CB, DS or ES equipment")
    else:
        _validate_signal_binding(mapping["source"], ieds)


def _validate_relay(data, ieds):
    seen = set()
    for entry in data.get("relay_statuses") or []:
        if entry["id"] in seen:
            raise StudioError("Relay status identifiers must be unique")
        seen.add(entry["id"])
        _binding_signal(entry["source"], ieds)


def _validate_control(data, ieds):
    binding = data.get("control_binding")
    if not binding:
        return
    if data["kind"] in DRAWING_ONLY:
        raise StudioError(f"{DRAWING_ONLY[data['kind']]} is a drawing point and does not accept a control binding")
    if data["kind"] not in CONTROLLABLE:
        raise StudioError("This equipment type does not accept a control binding")
    if data["kind"] == "PUSH_BUTTON" and data.get("push_mode") != "spc_momentary":
        raise StudioError("Only a momentary command push button accepts a command binding")
    ied = next((item for item in ieds if item["id"] == binding["ied_id"]), None)
    control = next((item for item in ied["controls"] if item["ref"] == binding["ref"]), None) if ied else None
    if control is None:
        raise StudioError("Equipment references an unknown control")
    expected = "DPC" if data["kind"] in SWITCHES else "SPC"
    if control.get("cdc") != expected:
        raise StudioError(f"{data['kind']} equipment requires a {expected} control")
    profile = binding.get("compatibility", "standard")
    if profile != "standard" and (data["kind"] not in SWITCHES or control.get("model") != 4):
        raise StudioError("CONCEN compatibility requires an enhanced-SBO DPC switch control")


def _settle_infeed(data):
    """Infeeds get their defaults; other kinds never carry (or store) infeed fields."""
    if data["kind"] == "INFEED":
        for key, default in INFEED_DEFAULTS.items():
            data[key] = data.get(key) or default
        return
    if any(data.get(key) is not None for key in INFEED_FIELDS):
        raise StudioError("Only an infeed accepts infeed settings")
    for key in INFEED_FIELDS:
        data.pop(key, None)


def validate_diagram(value, ieds):
    diagram = Diagram.model_validate(value).model_dump()
    ids = {node["id"] for node in diagram["nodes"]}
    if len(ids) != len(diagram["nodes"]):
        raise StudioError("Equipment identifiers must be unique")
    if len({edge["id"] for edge in diagram["edges"]}) != len(diagram["edges"]):
        raise StudioError("Wire identifiers must be unique")
    if any(edge["source"] not in ids or edge["target"] not in ids for edge in diagram["edges"]):
        raise StudioError("A wire references missing equipment")
    profiles = {}
    for node in diagram["nodes"]:
        data = node["data"]
        if data["kind"] != "RELAY" and data.get("relay_statuses"):
            raise StudioError("Only a relay accepts named status entries")
        if data["kind"] == "RELAY":
            _validate_relay(data, ieds)
        _settle_infeed(data)
        _validate_state(data, ieds)
        _validate_control(data, ieds)
        binding = data.get("control_binding")
        if binding:
            key = (binding["ied_id"], binding["ref"])
            profile = binding.get("compatibility", "standard")
            if key in profiles and profiles[key] != profile:
                raise StudioError("Equipment sharing one control must use the same compatibility profile")
            profiles[key] = profile
    return diagram


def validate_sources(sources, ieds):
    if not isinstance(sources, list) or len(sources) > 50000:
        raise StudioError("Invalid source definitions")
    signals = {(ied["id"], signal["ref"]): signal for ied in ieds for signal in ied["signals"]}
    result, seen = [], set()
    for item in sources:
        if not isinstance(item, dict):
            raise StudioError("Invalid source definition")
        key = (item.get("ied_id"), item.get("ref"))
        if key not in signals or key in seen:
            raise StudioError("Source references an unknown or duplicate signal")
        seen.add(key)
        result.append({"ied_id": key[0], "ref": key[1],
                       "spec": validate_source(signals[key], item.get("spec"))})
    return result
