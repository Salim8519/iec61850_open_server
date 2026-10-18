"""Pure rules for equipment test buttons on simulated IEDs (ADR-0067).

A test button writes the bound status or measurement point as if the plant changed by
itself; no command is sent. Each rule returns (binding, value, quality) writes.
"""
from .equipment_control import SWITCHES

# IEC 61850 Quality: validity bits 0-1, value 2 = invalid (as written by libiec61850).
QUALITY_GOOD, QUALITY_INVALID = 0, 2
DBPOS = {"moving": 0, "open": 1, "closed": 2, "bad": 3}
CONTACTS = {"open": (True, False), "closed": (False, True), "moving": (False, False), "bad": (True, True)}
MEASURED = {"CT", "VT"}
FED = {"SOURCE", "INFEED"}


def button_actions(data):
    """Test buttons offered for this equipment, by its kind and bound points."""
    kind, mapping = data["kind"], data.get("state_mapping")
    if kind == "RELAY":
        return ["on", "off"] if data.get("relay_statuses") else []
    if not mapping:
        return []
    if kind in SWITCHES:
        return ["open", "closed"] if mapping["mode"] == "single_to_dbpos" else ["open", "closed", "moving", "bad"]
    if kind == "TRIP_RELAY":
        return ["trip", "reset"]
    if kind in MEASURED:
        return ["set", "bad_quality"]
    if kind in FED:
        return ["live", "dead"]
    return []


def button_writes(data, action, value=None, row=None, kind_of=lambda binding: None):
    """Writes for one test action; `kind_of` gives the SCL kind of a bound point."""
    if action not in button_actions(data):
        raise ValueError("This test action does not fit this equipment")
    kind, mapping = data["kind"], data.get("state_mapping")
    if kind == "RELAY":
        entry = next((item for item in data["relay_statuses"] if item["id"] == row), None)
        if entry is None:
            raise ValueError("Choose a relay panel row")
        return [(entry["source"], action == "on", None)]
    if kind in SWITCHES:
        if mapping["mode"] == "direct":
            return [(mapping["source"], DBPOS[action], None)]
        if mapping["mode"] == "single_to_dbpos":
            return [(mapping["source"], action == "closed", None)]
        opened, closed = CONTACTS[action]
        return [(mapping["open"], opened, None), (mapping["closed"], closed, None)]
    source = mapping["source"]
    if kind == "TRIP_RELAY":
        operated = data.get("trip_state_polarity", "operated_true") == "operated_true"
        return [(source, operated if action == "trip" else not operated, None)]
    if kind in MEASURED:
        if action == "bad_quality":
            return [(source, None, QUALITY_INVALID)]
        if not isinstance(value, (int, float)) or isinstance(value, bool):
            raise ValueError("Enter a number")
        return [(source, value, QUALITY_GOOD)]
    live = action == "live"
    return [(source, (2 if live else 1) if kind_of(source) == "Dbpos" else live, None)]
