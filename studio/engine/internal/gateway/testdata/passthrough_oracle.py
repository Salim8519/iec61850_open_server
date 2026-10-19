"""Oracle for passthrough (protocol converter): map_points, device_value, remote_sample."""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import gateway_fixtures as fx  # noqa: E402
import master_fixtures as mx  # noqa: E402
import master_values as mv  # noqa: E402
from oracle_lib import rng, seed  # noqa: E402
from studio_app.domain import master_points as mp  # noqa: E402
from studio_app.domain import master_rules as mr  # noqa: E402
from studio_app.domain import passthrough as pt  # noqa: E402

r = rng(seed())
count = int(sys.argv[2]) if len(sys.argv) > 2 else 2000
PROTOCOLS = ["iec104", "dnp3", "modbus"]


def display_rows():
    """Rows exactly as the converter gets them: merged values shown by master_points.rows."""
    _, table = mv.merge_sequence(mv.merge_batches(r, odd=0))
    labels = {key: r.choice(["Pump", "", "  "]) for key in r.sample(sorted(table), min(len(table), 4))}
    unticked = r.sample(sorted(table), min(len(table), r.randint(0, 2)))
    return mp.rows(table, r.choice(PROTOCOLS), unticked, labels, r.random() < 0.8)


def raw_row():
    group = r.choice(mv.GROUP_NAMES)
    address = r.choice([r.randint(0, 30), 0, 1, True, False, 2 ** 70, -2 ** 70, -3, 16777215, 65535, 65534])
    row = {"key": f"{group}:{address}", "group": group, "address": address, "group_name": mp.GROUP_NAMES[group]}
    for key, values in (("label", ["", "Pump", None, 0, 5]), ("ticked", [True, False, 0, 1, None, "no"]),
                        ("type", ["uint16", "int32", "float32", None])):
        if r.random() < 0.7:
            row[key] = r.choice(values)
    return row


def random_routes():
    routes = [mx.random_route(r) for _ in range(r.randint(0, 4))]
    try:
        return mr.validate_routes(routes)
    except (ValueError, TypeError):
        return [{"upstream": r.choice([0, 1, 5, 16777215]), "upstream_kind": r.choice(["command_single", "command_double"]),
                 "kind": r.choice(["single", "double", "coil"]), "address": r.randint(0, 9), "select": r.random() < 0.5}]


for _ in range(count):
    if r.random() < 0.6:
        rows = display_rows()
    else:
        rows = [raw_row() for _ in range(r.randint(0, 12))]
    if r.random() < 0.02:
        rows = rows + [r.choice([{"group": "single"}, {"key": "k", "group": "bogus", "address": 1, "label": "x"},
                                 {"key": "k", "group": ["x"], "address": 1, "label": "x"}, 5, None])]
    protocol = r.choice(PROTOCOLS)
    routes = random_routes()
    fx.case("map_points", [rows, protocol, routes], lambda: pt.map_points(fx.fresh(rows), protocol, fx.fresh(routes)))
    route, value = {"kind": r.choice(["double", "single", "coil", "binary_output"])}, r.random() < 0.5
    fx.case("device_value", [route, value], lambda: pt.device_value(route, value))
    point = {"id": r.choice(["remote:single:1", 5]), "kind": r.choice(["single", "double", "integer", "float",
                                                                        "command_single"])}
    sample = None if r.random() < 0.1 else {k: v for k, v in mv.master_value(r, 0).items() if k != "group"}
    if sample is not None and r.random() < 0.15:
        sample["value"] = r.choice(["5", "x", " 1.5 ", "nan", [1], {"a": 1}, 10 ** 400, float("nan"), float("inf")])
    good, link = r.random() < 0.7, r.random() < 0.7
    fx.case("remote_sample", [point, sample, good, link], lambda: pt.remote_sample(point, fx.fresh(sample), good, link))
crowded = [{"key": f"single:{n}", "group": "single", "address": 16777215, "group_name": "Single point", "label": ""}
           for n in range(3)]
fx.case("map_points", [crowded, "iec104", []], lambda: pt.map_points(crowded, "iec104", []))
fx.case("remote_sample", [{"kind": "float"}, None, True, True], lambda: pt.remote_sample({"kind": "float"}, None, True, True))
fx.case("remote_sample", [{"id": "a"}, {"value": 1}, True, True], lambda: pt.remote_sample({"id": "a"}, {"value": 1}, True, True))
