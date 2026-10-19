"""Oracle for master_rules: protocol, host, settings, Modbus ranges, probe, routes, normalize."""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import gateway_fixtures as fx  # noqa: E402
import master_fixtures as mx  # noqa: E402
from oracle_lib import rng, seed  # noqa: E402
from studio_app.domain import master_rules as mr  # noqa: E402

r = rng(seed())
count = int(sys.argv[2]) if len(sys.argv) > 2 else 2000
PROTOCOL_VALUES = ["iec104", "dnp3", "modbus"] * 5 + fx.JUNK + ["IEC104", "iec 104"]
ROLE_VALUES = list(mr.ROLES) * 3 + fx.JUNK + ["Gateway"]


def random_host():
    if r.random() < 0.5:
        return r.choice(mx.HOSTS)
    alphabet = "ab9.-_ \téZ"
    return "".join(r.choice(alphabet) for _ in range(r.randint(0, 8)))


def random_span():
    if r.random() < 0.03:
        return r.choice([{"start": 1, "count": 1}, {"table": "coils", "start": 1}, {"table": "bogus", "count": 4},
                         {"table": ["x"], "count": 4}, {"table": 5, "count": 4}, "x", None])
    span = {"table": r.choice(mx.TABLE_NAMES[:4]), "start": r.choice([0, 7, 100, 65000, 65535]),
            "count": r.choice([0, 1, 2, 3, 4, 7, 124, 125, 126, 250, 251, 1999, 2000, 2001, 4500, -1, -5, True])}
    if r.random() < 0.03:
        span[r.choice(["start", "count"])] = r.choice([1.5, "x", None])
    return span


for _ in range(count):
    value = r.choice(PROTOCOL_VALUES)
    fx.case("protocol_arg", [value], lambda: mr.protocol_arg(fx.fresh(value)))
    host = random_host()
    fx.case("validate_host", [host], lambda: mr.validate_host(fx.fresh(host)))
    protocol, values = r.choice(PROTOCOL_VALUES), mx.random_settings(r)
    fx.case("validate_settings", [protocol, values], lambda: mr.validate_settings(fx.fresh(protocol), fx.fresh(values)))
    ranges = mx.random_list(r, mx.random_range if r.random() < 0.6 else mx.valid_range, 20)
    fx.case("validate_ranges", [ranges], lambda: mr.validate_ranges(fx.fresh(ranges)))
    probe = [r.choice(mx.TABLE_NAMES), r.choice(mx.EDGE_INTS + fx.JUNK + [9999, 10000, 10001]),
             r.choice(mx.EDGE_INTS + fx.JUNK + [9999, 10000, 10001])]
    fx.case("validate_probe", probe, lambda: mr.validate_probe(*fx.fresh(probe)))
    found = [random_span() for _ in range(r.randint(0, 4))]
    data_type = r.choice(mx.DATA_TYPE_NAMES[1:6] * 4 + mx.DATA_TYPE_NAMES)
    fx.case("ranges_from_probe", [found, data_type], lambda: mr.ranges_from_probe(fx.fresh(found), fx.fresh(data_type)))
    routes = mx.random_list(r, mx.random_route, 500)
    fx.case("validate_routes", [routes], lambda: mr.validate_routes(fx.fresh(routes)))
    raw = mx.random_master(r)
    fx.case("normalize_master", [raw], lambda: mr.normalize_master(fx.fresh(raw)))
    role = r.choice(ROLE_VALUES)
    fx.case("validate_role", [role], lambda: mr.validate_role(fx.fresh(role)))
fx.case("empty_master", [], mr.empty_master)
labels = {"k%05d" % number: "v" for number in range(20005)}
fx.case("normalize_master", [{"labels": {"dnp3": labels}, "unticked": {"modbus": list(labels)}}],
        lambda: mr.normalize_master({"labels": {"dnp3": labels}, "unticked": {"modbus": list(labels)}}))
