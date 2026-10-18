"""Protocol converter mapping: device points -> upstream points (ADR-0061, spec MASTER-09..11)."""
import pytest
from studio_app.domain.master_rules import validate_routes
from studio_app.domain.passthrough import device_value, map_points, remote_sample


def row(group, address, **extra):
    return {"key": f"{group}:{address}", "group": group, "group_name": group.title(), "address": address,
            "ticked": True, "label": "", **extra}


def test_watched_points_keep_their_address_or_move_to_the_next_free_one():
    rows = [row("binary", 0), row("analog", 0), row("binary_output", 4), row("double_binary", 7, ticked=False)]
    points, notes = map_points(rows, "iec104")
    by_id = {point["id"]: point for point in points}
    assert by_id["remote:binary:0"]["address"] == 1 and by_id["remote:binary:0"]["kind"] == "single"
    assert by_id["remote:analog:0"]["address"] == 2 and by_id["remote:analog:0"]["kind"] == "float"
    assert by_id["remote-command:binary_output:4"]["route"] == {"kind": "binary_output", "address": 4}
    assert "remote:double_binary:7" not in by_id  # unticked points are not served
    assert any("served at address 2" in note for note in notes)
    # change 0089: IEC 104 has no object address 0; it is not "taken"
    assert "Binary 0 is served at address 1 (IEC 104 has no address 0)" in notes
    assert not any("was taken" in note for note in notes)
    dnp3, notes = map_points(rows[:2], "dnp3")
    assert [point["address"] for point in dnp3] == [0, 0] and notes == []  # different DNP3 index spaces


def test_command_routes_need_a_free_upstream_address():
    routes = validate_routes([{"upstream": 7001, "upstream_kind": "command_double", "kind": "double", "address": 6001},
                              {"upstream": 1, "upstream_kind": "command_single", "kind": "single", "address": 6002}])
    points, notes = map_points([row("single", 1)], "iec104", routes)
    route = next(point for point in points if point["id"] == "route:1")
    assert route["address"] == 7001 and route["route"] == {"kind": "double", "address": 6001, "select": True}
    assert not any(point["id"] == "route:2" for point in points) and "already used" in notes[0]
    with pytest.raises(ValueError, match="single, double, binary_output or coil"):
        validate_routes([{"upstream": 1, "upstream_kind": "command_single", "kind": "setpoint", "address": 1}])
    assert device_value({"kind": "double"}, True) == 2 and device_value({"kind": "coil"}, False) == 0


def test_remote_values_are_invalid_upstream_while_the_device_is_bad_or_away():
    point = {"id": "remote:analog:2", "kind": "float"}
    good = remote_sample(point, {"value": 230.5, "time_ms": 5}, True, True)
    assert good == {"id": "remote:analog:2", "value": 230.5, "quality": 0, "time_ms": 5, "source_ok": True}
    assert remote_sample(point, {"value": 1.0}, False, True)["quality"] == 2
    assert remote_sample(point, {"value": 1.0}, True, False)["source_ok"] is False
    assert remote_sample(point, None, True, True)["source_ok"] is False
