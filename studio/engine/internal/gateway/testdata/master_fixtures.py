"""Random settings, Modbus ranges, routes and master sections for the oracle scripts."""
from gateway_fixtures import JUNK

SETTING_NAMES = ["port", "common_address", "originator_address", "t0", "t1", "t2", "t3", "k", "w",
                 "time_tagged_commands", "gi_minutes", "outstation_address", "master_address",
                 "integrity_minutes", "event_seconds", "unsolicited", "unit_id", "poll_ms", "bogus", ""]
EDGE_INTS = [0, 1, 2, 7, 8, 10, 12, 15, 20, 199, 200, 254, 255, 256, 1440, 1441, 3600, 3601, 32767, 32768,
             60000, 60001, 65519, 65520, 65534, 65535, 65536, -1]
TABLE_NAMES = ["coils", "discrete_inputs", "holding_registers", "input_registers", "registers", None, 5,
               ["coils"], {"a": 1}]
DATA_TYPE_NAMES = ["bool", "uint16", "int16", "uint32", "int32", "float32", "float", None, 3, ["bool"]]
HOSTS = ["", "192.168.1.20", " rtu1.station.local ", "-bad", "bad-", "a" * 253, "a" * 254, "a", "a.", ".a",
         "a-b", "a b", "hést", "a_b", "1.2.3.4\n", None, 5, True, ["x"], "\tlocalhost\n", "A9", "a..b"]


def setting_value(r):
    if r.random() < 0.6:
        return r.choice(EDGE_INTS)
    return r.choice(JUNK + [True, False])


def random_settings(r):
    choice = r.random()
    if choice < 0.1:
        return r.choice([None, {}, [], "", 0, False, [1], "x", 5, True, 1.5])
    names = r.sample(SETTING_NAMES, r.randint(0, 4))
    if r.random() < 0.5:  # mostly known keys of one protocol, so the checks after the loop run
        names = r.sample([["port", "t1", "t2", "k", "w"], ["port", "unit_id", "poll_ms"],
                          ["outstation_address", "master_address", "unsolicited"]][r.randint(0, 2)], 2)
    return {name: setting_value(r) for name in names}


def random_range(r):
    if r.random() < 0.05:
        return r.choice(JUNK)
    raw = {}
    if r.random() < 0.95:
        raw["table"] = r.choice(TABLE_NAMES[:4] * 4 + TABLE_NAMES)
    if r.random() < 0.6:
        raw["data_type"] = r.choice(DATA_TYPE_NAMES[:6] * 3 + DATA_TYPE_NAMES)
    for key in ("start", "count"):
        if r.random() < 0.95:
            raw[key] = r.choice([r.randint(0, 65535), r.randint(1, 130), r.randint(1, 2000)] * 3 + EDGE_INTS + JUNK)
    if r.random() < 0.4:
        raw["word_order"] = r.choice(["big", "little", "middle", None, 1])
    return raw


def random_route(r):
    if r.random() < 0.05:
        return r.choice(JUNK)
    raw = {}
    for key, values in (("kind", ["single", "double", "binary_output", "coil", "analog", None, ["coil"]]),
                        ("upstream_kind", ["command_single", "command_double", "single", None])):
        if r.random() < 0.95:
            raw[key] = r.choice(values[:4] * 4 + values)
    for key in ("upstream", "address"):
        if r.random() < 0.95:
            raw[key] = r.choice([r.randint(0, 300), r.randint(0, 16777215)] * 4 + [16777216, -1] + JUNK)
    if r.random() < 0.6:
        raw["select"] = r.choice([True, False, 1, None, "yes"])
    return raw


def random_list(r, maker, limit):
    if r.random() < 0.05:
        return r.choice(JUNK)
    if r.random() < 0.03:
        return [maker(r) for _ in range(limit + 1)]
    return [maker(r) for _ in range(r.randint(0, 4))]


def random_label_map(r):
    choice = r.random()
    if choice < 0.15:
        return r.choice(JUNK)
    labels = {}
    for _ in range(r.randint(0, 5)):
        key = r.choice(["single:1", "double:2", "k" * 40, "k" * 41, "é" * 40, "coil:0", ""])
        labels[key] = r.choice(["Breaker", " ", "", "x" * 90, "\U0001F600" * 85, 5, None, ["a"]])
    return labels


def per_protocol(r, maker):
    choice = r.random()
    if choice < 0.1:
        return r.choice([None, {}, [], [1], "x", 5])
    return {name: maker(r) for name in r.sample(["iec104", "dnp3", "modbus", "other"], r.randint(0, 4))}


def random_unticked(r):
    return r.choice([[r.choice(["single:1", "coil:5", 5, None, ["x"]]) for _ in range(r.randint(0, 4))],
                     ["a", "b"], {"single:1": 1}, "abc", 5, None, [], "", True])


def valid_range(r):
    table = r.choice(TABLE_NAMES[:4])
    bits = table in ("coils", "discrete_inputs")
    data_type = "bool" if bits else r.choice(DATA_TYPE_NAMES[1:6])
    return {"table": table, "data_type": data_type, "start": r.randint(0, 60000), "count": r.randint(1, 60),
            "word_order": r.choice(["big", "little"])}


def valid_master(r):
    """A master section that passes every check (so the success path is compared too)."""
    return {"protocol": r.choice(["iec104", "dnp3", "modbus"]), "host": r.choice(["", "10.0.0.5", " rtu.local "]),
            "auto_connect": r.choice([True, False]),
            "settings": {"iec104": {"port": r.randint(1, 65535), "t1": 30, "t2": r.randint(1, 29), "k": 12,
                                    "w": r.randint(1, 12), "time_tagged_commands": r.choice([True, False])},
                         "dnp3": {"unsolicited": r.choice([True, False]), "event_seconds": r.randint(0, 3600)},
                         "modbus": {"poll_ms": r.randint(200, 60000)}},
            "unticked": {"iec104": ["single:1", 5, "double:3"], "dnp3": [], "modbus": ["coil:0"]},
            "labels": {"iec104": {"single:1": "Breaker A", "double:3": "  "}, "modbus": {"coil:0": "Pump"}},
            "modbus_ranges": [valid_range(r) for _ in range(r.randint(0, 3))],
            "routes": [{"upstream": r.randint(0, 99), "upstream_kind": "command_double", "kind": "double",
                        "address": r.randint(0, 99), "select": r.choice([True, False])}]}


def random_master(r):
    if r.random() < 0.35:
        return valid_master(r)
    if r.random() < 0.08:
        return r.choice([None, [], "x", 5, [1]])
    raw = {}
    if r.random() < 0.7:
        raw["protocol"] = r.choice(["iec104", "dnp3", "modbus"] * 6 + ["x", None, ["iec104"], "IEC104"])
    if r.random() < 0.6:
        raw["host"] = r.choice(HOSTS)
    if r.random() < 0.5:
        raw["auto_connect"] = r.choice([True, False, 1, "yes", None])
    for key, maker in (("settings", random_settings), ("unticked", random_unticked), ("labels", random_label_map)):
        if r.random() < 0.6:
            raw[key] = per_protocol(r, maker)
    if r.random() < 0.6:
        raw["modbus_ranges"] = random_list(r, random_range, 20)
    if r.random() < 0.6:
        raw["routes"] = random_list(r, random_route, 500)
    return raw
