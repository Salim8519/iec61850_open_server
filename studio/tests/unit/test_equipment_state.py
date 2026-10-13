"""Constructed switch states follow the IEC 61850 Dbpos truth table."""
from studio_app.domain.equipment_state import binding_keys, evaluate_state


OPEN = {"ied_id": "first", "ref": "open", "report_ref": "first/LLN0.BR.Events"}
CLOSED = {"ied_id": "second", "ref": "closed", "report_ref": None}


def value_reader(values):
    return lambda key: values.get(key)


def test_two_contact_mapping_preserves_all_four_states():
    mapping = {"mode": "contacts_to_dbpos", "open": OPEN, "closed": CLOSED}
    assert binding_keys(mapping) == [("first", "open"), ("second", "closed")]
    cases = [
        ({("first", "open"): False, ("second", "closed"): False}, 0),
        ({("first", "open"): True, ("second", "closed"): False}, 1),
        ({("first", "open"): False, ("second", "closed"): True}, 2),
        ({("first", "open"): True, ("second", "closed"): True}, 3),
    ]
    assert [evaluate_state(mapping, value_reader(values)) for values, _ in cases] == [
        expected for _, expected in cases]


def test_single_boolean_and_missing_values_are_not_guessed():
    mapping = {"mode": "single_to_dbpos", "source": OPEN}
    assert evaluate_state(mapping, value_reader({("first", "open"): False})) == 1
    assert evaluate_state(mapping, value_reader({("first", "open"): True})) == 2
    assert evaluate_state(mapping, value_reader({})) is None
    contacts = {"mode": "contacts_to_dbpos", "open": OPEN, "closed": CLOSED}
    assert evaluate_state(contacts, value_reader({("first", "open"): True})) is None


def test_direct_mapping_returns_the_native_value_unchanged():
    mapping = {"mode": "direct", "source": OPEN}
    assert evaluate_state(mapping, value_reader({("first", "open"): 3})) == 3
