"""Equipment test buttons write the bound point only (ADR-0067, change 0097)."""
import pytest
from studio_app.domain.equipment_test import QUALITY_INVALID, button_actions, button_writes

POINT = {"ied_id": "a", "ref": "LD/XCBR1.Pos.stVal", "report_ref": None}


def switch(mode="direct"):
    if mode == "contacts_to_dbpos":
        return {"kind": "CB", "state_mapping": {"mode": mode, "open": dict(POINT, ref="o"), "closed": dict(POINT, ref="c")}}
    return {"kind": "CB", "state_mapping": {"mode": mode, "source": POINT}}


def test_switch_positions_write_dbpos():
    assert [button_writes(switch(), action)[0][1] for action in ("open", "closed", "moving", "bad")] == [1, 2, 0, 3]


def test_contacts_and_single_points():
    assert [value for _, value, _ in button_writes(switch("contacts_to_dbpos"), "moving")] == [False, False]
    assert button_actions(switch("single_to_dbpos")) == ["open", "closed"]
    assert button_writes(switch("single_to_dbpos"), "closed")[0][1] is True


def test_no_binding_means_no_buttons():
    assert button_actions({"kind": "CB"}) == []
    assert button_actions({"kind": "JUNCTION", "state_mapping": {"mode": "direct", "source": POINT}}) == []
    with pytest.raises(ValueError):
        button_writes({"kind": "DS"}, "open")


def test_relay_rows_trip_and_measurements():
    relay = {"kind": "RELAY", "relay_statuses": [{"id": "r1", "label": "Trip", "source": POINT}]}
    assert button_writes(relay, "on", row="r1") == [(POINT, True, None)]
    with pytest.raises(ValueError, match="row"):
        button_writes(relay, "off", row="missing")
    trip = {"kind": "TRIP_RELAY", "trip_state_polarity": "operated_false", "state_mapping": {"mode": "direct", "source": POINT}}
    assert button_writes(trip, "trip")[0][1] is False
    ct = {"kind": "CT", "state_mapping": {"mode": "direct", "source": POINT}}
    assert button_writes(ct, "set", 12.5) == [(POINT, 12.5, 0)]
    assert button_writes(ct, "bad_quality") == [(POINT, None, QUALITY_INVALID)]
    with pytest.raises(ValueError, match="number"):
        button_writes(ct, "set", "x")


def test_source_follows_point_kind():
    source = {"kind": "INFEED", "state_mapping": {"mode": "direct", "source": POINT}}
    assert button_writes(source, "live", kind_of=lambda _: "Dbpos")[0][1] == 2
    assert button_writes(source, "dead", kind_of=lambda _: "BOOLEAN")[0][1] is False
