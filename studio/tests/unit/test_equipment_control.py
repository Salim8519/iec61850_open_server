"""Type-driven equipment commands produce deterministic feedback values."""
import pytest
from studio_app.domain.equipment_control import expected_cdc, feedback_targets, semantic_command


SOURCE = {"ied_id": "ied", "ref": "status", "report_ref": None}
OPEN = {"ied_id": "ied", "ref": "open", "report_ref": None}
CLOSED = {"ied_id": "ied", "ref": "closed", "report_ref": None}


def test_control_cdc_comes_from_equipment_kind_not_reference_names():
    assert [expected_cdc(kind) for kind in ("CB", "DS", "ES")] == ["DPC"] * 3
    assert expected_cdc("TRIP_RELAY") == "SPC"
    with pytest.raises(ValueError, match="not controllable"):
        expected_cdc("BUS")


def test_switch_feedback_supports_native_single_and_constructed_states():
    direct = {"mode": "direct", "source": SOURCE}
    single = {"mode": "single_to_dbpos", "source": SOURCE}
    contacts = {"mode": "contacts_to_dbpos", "open": OPEN, "closed": CLOSED}
    assert feedback_targets({"kind": "CB"}, direct, "open") == [(SOURCE, 1)]
    assert feedback_targets({"kind": "DS"}, direct, "close") == [(SOURCE, 2)]
    assert feedback_targets({"kind": "ES"}, single, "open") == [(SOURCE, False)]
    assert feedback_targets({"kind": "CB"}, single, "close") == [(SOURCE, True)]
    assert feedback_targets({"kind": "DS"}, contacts, "open") == [(OPEN, True), (CLOSED, False)]
    assert feedback_targets({"kind": "DS"}, contacts, "close") == [(OPEN, False), (CLOSED, True)]


def test_trip_relay_operates_and_resets_one_boolean_state():
    direct = {"mode": "direct", "source": SOURCE}
    operated_high = {"kind": "TRIP_RELAY", "trip_state_polarity": "operated_true"}
    operated_low = {"kind": "TRIP_RELAY", "trip_state_polarity": "operated_false"}
    assert feedback_targets(operated_high, direct, "operate") == [(SOURCE, True)]
    assert feedback_targets(operated_high, direct, "reset") == [(SOURCE, False)]
    assert feedback_targets(operated_low, direct, "operate") == [(SOURCE, False)]
    assert feedback_targets({"kind": "CB"}, None, "close") == []
    with pytest.raises(ValueError, match="Open or Close"):
        feedback_targets({"kind": "CB"}, direct, "operate")


def test_semantic_actions_resolve_trip_polarity_and_push_value():
    low = {"kind": "TRIP_RELAY", "trip_command_polarity": "operate_false"}
    high = {"kind": "TRIP_RELAY", "trip_command_polarity": "operate_true"}
    assert semantic_command(low, "operate") == ("operate", False)
    assert semantic_command(low, "reset") == ("reset", True)
    assert semantic_command(high, "operate") == ("operate", True)
    push = {"kind": "PUSH_BUTTON", "push_mode": "spc_momentary", "push_value": False}
    assert semantic_command(push, "press") == ("press", False)
    rest_low = {"kind": "PUSH_BUTTON", "push_mode": "self_resetting", "push_rest_value": False}
    rest_high = {"kind": "PUSH_BUTTON", "push_mode": "self_resetting", "push_rest_value": True}
    assert semantic_command(rest_low, "press") == ("press", True)
    assert semantic_command(rest_high, "press") == ("press", False)
