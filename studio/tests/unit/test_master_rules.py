"""SCADA master settings, discovered point table and the format 13 -> 14 upgrade (ADR-0061)."""
import pytest
from studio_app.domain.gateway_rules import normalize_gateway
from studio_app.domain.master_points import export_csv, import_labels, merge, rows
from studio_app.domain.master_rules import (normalize_master, ranges_from_probe, validate_host, validate_probe,
                                              validate_ranges, validate_settings)


def test_master_settings_use_defaults_and_explain_every_problem():
    assert validate_settings("iec104", {"common_address": 7})["gi_minutes"] == 15
    with pytest.raises(ValueError, match="t2 must be shorter than t1"):
        validate_settings("iec104", {"t1": 5, "t2": 10})
    with pytest.raises(ValueError, match="Unit id must be a whole number from 0 to 255"):
        validate_settings("modbus", {"unit_id": 300})
    with pytest.raises(ValueError, match="Choose IEC 104"):
        validate_settings("iec101", {})
    assert validate_host(" 10.0.0.5 ") == "10.0.0.5" and validate_host("rtu1.station.local")
    with pytest.raises(ValueError, match="device address"):
        validate_host("10.0.0.5; drop")


def test_modbus_ranges_are_bounded_per_request():
    clean = validate_ranges([{"table": "holding_registers", "start": 0, "count": 62, "data_type": "float32"}])
    assert clean[0]["word_order"] == "big"
    with pytest.raises(ValueError, match="read 1 to 62 values"):
        validate_ranges([{"table": "holding_registers", "start": 0, "count": 63, "data_type": "float32"}])
    with pytest.raises(ValueError, match="registers as uint16"):
        validate_ranges([{"table": "coils", "start": 0, "count": 5, "data_type": "uint16"}])
    with pytest.raises(ValueError, match="at most 20"):
        validate_ranges([{"table": "coils", "start": 0, "count": 1}] * 21)


def test_format_13_gateway_sections_upgrade_to_station_gateway_role():
    old = {"protocol": "dnp3", "listen": "this_pc", "points": {}}
    upgraded = normalize_gateway(old)
    assert upgraded["role"] == "gateway" and upgraded["protocol"] == "dnp3"
    assert upgraded["master"] == normalize_master(None) and upgraded["master"]["host"] == ""
    with pytest.raises(ValueError, match="SCADA master"):
        normalize_gateway({**old, "role": "slave"})


def test_discovered_points_merge_word_quality_and_keep_labels():
    table = {}
    added = merge(table, [{"group": "double", "type": "M_DP_NA_1", "address": 2001, "value": 2, "quality": 0,
                           "cause": "interrogated"},
                          {"group": "float", "type": "M_ME_NC_1", "address": 3001, "value": 1.5, "quality": 0xC0,
                           "cause": "interrogated"},
                          {"group": "nonsense", "address": 1}])
    assert added == ["double:2001", "float:3001"]
    merge(table, [{"group": "double", "address": 2001, "value": 2, "quality": 0, "cause": "interrogated"}])
    assert table["double:2001"]["changes"] == 0
    merge(table, [{"group": "double", "address": 2001, "value": 1, "quality": 0, "cause": "spontaneous"}])
    shown = rows(table, "iec104", ["float:3001"], {"double:2001": "Feeder breaker"}, link_open=True)
    assert [row["key"] for row in shown] == ["double:2001", "float:3001"]
    assert shown[0]["value_text"] == "open (OFF)" and shown[0]["changes"] == 1 and shown[0]["label"] == "Feeder breaker"
    assert shown[1]["quality_text"] == "invalid, not topical" and shown[1]["ticked"] is False
    lost = rows(table, "iec104", [], {}, link_open=False)[0]
    assert lost["stale"] is True and lost["good"] is False  # a last-known value is never "good"
    assert lost["quality_text"].startswith("not current (link lost)")
    text = export_csv(shown)
    labels, problems = import_labels(text.replace("Feeder breaker", "Bay 2 breaker"))
    assert labels == {"double:2001": "Bay 2 breaker"} and problems == []
    assert import_labels("a,b\n1,2")[1][0].startswith("The CSV needs")


def test_find_registers_is_bounded_and_found_spans_become_read_ranges():
    assert validate_probe("holding_registers", 0, 9999) == ("holding_registers", 0, 9999)
    with pytest.raises(ValueError, match="1 to 10000 addresses"):
        validate_probe("holding_registers", 0, 10000)
    with pytest.raises(ValueError, match="Choose coils"):
        validate_probe("registers", 0, 1)
    found = [{"table": "holding_registers", "start": 0, "count": 300}, {"table": "coils", "start": 10, "count": 3}]
    ranges = ranges_from_probe(found, "float32")
    assert [(item["start"], item["count"], item["data_type"]) for item in ranges] == [
        (0, 62, "float32"), (124, 62, "float32"), (248, 26, "float32"), (10, 3, "bool")]
    assert validate_ranges(ranges) == ranges
    with pytest.raises(ValueError, match="Read registers as"):
        ranges_from_probe(found, "bool")


def test_dnp3_counters_use_dnp3_words_and_frozen_counters_are_kept():
    """Found against the FreyrSCADA DNP3 outstation (change 0089): frozen counters were dropped
    and DNP3 counters were called by the IEC 104 name "Integrated total"."""
    table = {}
    added = merge(table, [{"group": "counter", "type": "Group20Var1", "address": 0, "value": 3, "quality": 1,
                           "cause": "integrity poll"},
                          {"group": "frozen_counter", "type": "Group21Var1", "address": 0, "value": 7, "quality": 1,
                           "cause": "integrity poll"}])
    assert added == ["counter:0", "frozen_counter:0"]
    names = {row["key"]: row["group_name"] for row in rows(table, "dnp3", [], {}, link_open=True)}
    assert names == {"counter:0": "Counter", "frozen_counter:0": "Frozen counter"}
    assert rows(table, "iec104", [], {}, link_open=True)[0]["group_name"] == "Integrated total"


def test_dnp3_state_bits_are_not_quality():
    """Change 0089: an open DNP3 double-bit input (state 01 in bits 6-7) read as "reference error"."""
    from studio_app.domain.master_points import quality_words
    assert quality_words("dnp3", 0x01 | 0x40, "double_binary") == []      # DETERMINED_OFF, online
    assert quality_words("dnp3", 0x01 | 0x80, "binary") == []             # state ON
    assert quality_words("dnp3", 0x01 | 0x80, "binary_output") == []
    assert quality_words("dnp3", 0x01 | 0x40, "analog") == ["reference error"]
    assert quality_words("dnp3", 0x01 | 0x20, "analog") == ["over range"]
    assert quality_words("dnp3", 0x01 | 0x20 | 0x40, "counter") == ["rollover", "discontinuity"]
    assert quality_words("dnp3", 0x02, "binary_output") == ["offline", "restart"]
