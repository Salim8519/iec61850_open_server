"""normalize_gateway / normalize_master vectors and preference/list-filter vectors."""
import json

GATEWAYS = [None, {}, [], "x", 5, {"protocol": "modbus"}, {"protocol": "x"}, {"protocol": 1.5}, {"protocol": None},
            {"listen": "network"}, {"listen": "x"}, {"listen": ["network"]}, {"role": "master"}, {"role": "x"},
            {"real_commands": False, "auto_start": True}, {"real_commands": 1, "auto_start": "yes"},
            {"settings": None}, {"settings": []}, {"settings": [1]}, {"settings": {"iec104": None}},
            {"settings": {"iec104": [1]}}, {"settings": {"iec104": {"port": 2405, "t1": 20, "t2": 19}}},
            {"settings": {"iec104": {"t1": 10}}}, {"settings": {"iec104": {"w": 13}}}, {"settings": {"iec104": {"x": 1}}},
            {"settings": {"iec104": {"common_address": 0}}}, {"settings": {"iec104": {"k": True}}},
            {"settings": {"iec104": {"t3": 3600.0}}}, {"settings": {"modbus": {"unit_id": 256}}},
            {"settings": {"dnp3": {"unsolicited": 1}}}, {"settings": {"dnp3": {"unsolicited": False, "master_address": 65519}}},
            {"settings": {"dnp3": {"outstation_address": 10 ** 20}}},
            {"points": None}, {"points": [1]}, {"points": {"iec104": None}}, {"points": {"iec104": {}}},
            {"points": {"iec104": [{"kind": "single", "address": 1}]}},
            {"points": {"iec104": [{"kind": "single", "address": True, "ied": 5, "ref": None}]}},
            {"points": {"iec104": [{"kind": "single", "address": 1.0}]}}, {"points": {"iec104": [{"kind": "x", "address": 1}]}},
            {"points": {"iec104": [5]}}, {"points": {"modbus": [{"kind": "command_double", "address": 4, "id": "keep"}]}},
            {"points": {"dnp3": {"@repeat": {"kind": "float", "address": 1}, "count": 5001}}},
            {"points": {"dnp3": {"@repeat": {"kind": "float", "address": 1}, "count": 5000}}},
            {"master": []}, {"master": {"protocol": "x"}}, {"protocol": "x", "master": []}]
MASTERS = [None, {}, [], {"protocol": "dnp3"}, {"protocol": "modbus", "host": " rtu1.station.local "},
           {"host": "-bad"}, {"host": 5}, {"host": ""}, {"host": "a"}, {"host": "a" * 254}, {"host": "a" * 253},
           {"auto_connect": True}, {"auto_connect": 1}, {"settings": {"iec104": {"time_tagged_commands": 1}}},
           {"settings": {"iec104": {"time_tagged_commands": True, "gi_minutes": 0, "t0": 256}}},
           {"settings": {"dnp3": {"integrity_minutes": 1441}}}, {"settings": {"modbus": {"poll_ms": 199}}},
           {"settings": {"modbus": {"poll_ms": 200}}}, {"settings": {"dnp3": {"unsolicited": "x"}}},
           {"unticked": None}, {"unticked": []}, {"unticked": {"iec104": ["a", 1, None, "b"]}},
           {"unticked": {"iec104": "abc"}}, {"unticked": {"iec104": {"k": 1}}}, {"unticked": {"iec104": 5}},
           {"unticked": {"iec104": None}}, {"labels": [1]}, {"labels": {"iec104": [1]}},
           {"labels": {"iec104": {"a": "x" * 90, "b": " ", "c": 5, "d" * 41: "y", "e": "ok"}}},
           {"modbus_ranges": None}, {"modbus_ranges": [1]}, {"modbus_ranges": [{}] * 21},
           {"modbus_ranges": [{"table": "coils", "start": 0, "count": 2000}]},
           {"modbus_ranges": [{"table": "coils", "start": 0, "count": 2001}]},
           {"modbus_ranges": [{"table": "coils", "data_type": "uint16", "start": 0, "count": 1}]},
           {"modbus_ranges": [{"table": "holding_registers", "data_type": "float32", "start": 65500, "count": 18}]},
           {"modbus_ranges": [{"table": "holding_registers", "data_type": "float32", "start": 65500, "count": 19}]},
           {"modbus_ranges": [{"table": "input_registers", "start": -1, "count": 1}]},
           {"modbus_ranges": [{"table": "input_registers", "start": 0, "count": 126}]},
           {"modbus_ranges": [{"table": "input_registers", "start": 0, "count": 1, "word_order": "little"}]},
           {"modbus_ranges": [{"table": "input_registers", "start": 0, "count": 1, "word_order": "x"}]},
           {"modbus_ranges": [{"table": ["coils"], "start": 0, "count": 1}]},
           {"modbus_ranges": [{"table": "coils", "data_type": ["bool"], "start": 0, "count": 1}]},
           {"routes": None}, {"routes": [1]}, {"routes": [{"kind": ["coil"]}]}, {"routes": [{"kind": "coil"}]},
           {"routes": [{"kind": "coil", "upstream_kind": "command_single", "upstream": 1, "address": 2}]},
           {"routes": [{"kind": "double", "upstream_kind": "command_double", "upstream": 1, "address": True}]},
           {"routes": [{"kind": "double", "upstream_kind": "command_double", "upstream": 16777216, "address": 1}]},
           {"routes": [{"kind": "single", "upstream_kind": "command_single", "upstream": 0, "address": 0,
                        "select": False, "x": 1}]}]
STORED = [None, "", "{", "[]", "5", "{}", '{"tour_seen_version": 3, "sld_goose_layer": true, "sld_readers_layer": 1.0}',
          '{"tour_seen_version": 99999999999999999999999, "list_filters": {"ieds": {"q": "x", "Bad": 1, "on": true}}}',
          json.dumps({"list_filters": {"a-b": {"k": "v"}, "Bad": {}, "x\n": {"k\n": "v"}, "y": [], "z": {"t": "x" * 121}}}),
          json.dumps({"list_filters": {f"l{i}": {} for i in range(45)}}),
          json.dumps({"list_filters": {"l": {f"k{i}": True for i in range(20)}}})]
CHANGES = [None, [], "x", {}, {"tour_seen_version": 1}, {"tour_seen_version": 1001}, {"tour_seen_version": -1},
           {"tour_seen_version": True}, {"tour_seen_version": 1.0}, {"sld_goose_layer": 1}, {"sld_goose_layer": 2},
           {"zeta": 1, "alpha": 2}, {"list_filters": {"ieds": {"kind": "relay", "live": True}}},
           {"list_filters": []}, {"list_filters": {"Bad": {}}}, {"list_filters": {"ok": {"k": 5}}},
           {"list_filters": {"ok": []}}, {"list_filters": {"ok": {f"k{i}": True for i in range(17)}}},
           {"list_filters": {f"l{i}": {} for i in range(41)}}, {"list_filters": {"ok\n": {"k\n": "é"}}},
           {"sld_readers_layer": 1, "list_filters": {"a": {}}, "tour_seen_version": 1}]


def cases(fixtures):
    out = []

    def add(name, fn, **args):
        out.append({"name": name, "fn": fn, "args": args, "ieds": []})

    add("empty gateway", "empty_gateway")
    for index, raw in enumerate(GATEWAYS):
        add(f"gateway {index}", "normalize_gateway", raw=raw)
    for index, raw in enumerate(MASTERS):
        add(f"master {index}", "normalize_master", raw=raw)
        add(f"gateway master {index}", "normalize_gateway", raw={"master": raw})
    for index, text in enumerate(STORED):
        add(f"preferences get {index}", "preferences_get", stored_text=text)
        for number, changes in enumerate(CHANGES):
            if index in (0, 6, 7, 8) or number in (4, 12):
                add(f"preferences update {index} {number}", "preferences_update", stored_text=text, changes=changes)
    for index, stored in enumerate([None, [], {"a": {"k": "v"}}, {"a": [], "b": {"k": 1}},
                                    {f"l{i}": {"k": True} for i in range(45)}]):
        add(f"list filters clean {index}", "list_filters_clean", stored=stored)
    for index, changes in enumerate([{}, None, {"Bad": {}}, {"b": {"k": "v"}}, {"a": {"x": None}}]):
        add(f"list filters merge {index}", "list_filters_merge", current={"a": {"k": "v"}}, changes=changes)
    return out
