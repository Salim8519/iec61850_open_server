"""Oracle for gateway_rules: settings, point validation, listen address, normalize_gateway."""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import gateway_fixtures as fx  # noqa: E402
import master_fixtures as mx  # noqa: E402
from oracle_lib import rng, seed  # noqa: E402
from studio_app.domain import gateway_points as gp  # noqa: E402
from studio_app.domain import gateway_rules as gr  # noqa: E402

r = rng(seed())
count = int(sys.argv[2]) if len(sys.argv) > 2 else 2000
PROTOCOL_ARGS = list(gp.PROTOCOLS) * 5 + ["iec61850", "IEC104", ""]


def mutate_point(point, ieds, previous):
    point = dict(point)
    choice = r.random()
    if choice < 0.1:
        point["address"] = r.choice(fx.JUNK + [r.randint(-5, 70000), 16777215, 16777216, 65535, 65536, True])
    elif choice < 0.2:
        point["scale"] = r.choice(fx.JUNK + [2, -1.5, 1e308, 10 ** 400, 10 ** 20])
    elif choice < 0.27:
        point["kind"] = r.choice(fx.JUNK + list(gp.KINDS))
    elif choice < 0.33:
        ied = r.choice(ieds)
        point["ref"] = r.choice([fx.random_ref(r), r.choice(ied["signals"])["ref"], 5, None])
    elif choice < 0.38:
        point["ied"] = r.choice(fx.JUNK + ["ied99"] + [ied["id"] for ied in ieds])
    elif choice < 0.43:
        point["description"] = r.choice(fx.TEXTS + fx.JUNK)
    elif choice < 0.48:
        point.pop(r.choice(list(point)))
    elif choice < 0.56 and isinstance(previous, dict) and type(previous.get("address")) is int:
        point["address"] = previous["address"] + r.choice([0, 1, -1])
    return point


def random_points(ieds, protocol):
    if r.random() < 0.04:
        return r.choice(fx.JUNK)
    proposal = gp.propose(fx.fresh(ieds), protocol)[0] if ieds else []
    points, previous = [], None
    for point in r.sample(proposal, min(len(proposal), r.randint(0, 40))):
        if r.random() < 0.05:
            points.append(r.choice(fx.JUNK))
        previous = mutate_point(point, ieds, previous) if r.random() < 0.6 else point
        points.append(previous)
    return points


def stored_point():
    if r.random() < 0.03:
        return r.choice(fx.JUNK)
    point = {}
    for key, values in (("id", ["p1", 5, None]), ("kind", list(gp.KINDS) * 3 + ["analog", None, 5]),
                        ("address", [1, 0, -4, True, 2 ** 70, 1.5, None, "7"] + list(range(10))),
                        ("ied", ["ied1", 5, None, ["x"]]), ("ref", ["LD0/XCBR1.Pos.stVal", 2.5, None]),
                        ("scale", [1.0, 2]), ("extra", ["x"])):
        if r.random() < (0.97 if key in ("kind", "address") else 0.5):
            point[key] = r.choice(values)
    return point


def random_gateway():
    if r.random() < 0.06:
        return r.choice(fx.JUNK)
    raw = {}
    for key, values in (("protocol", list(gp.PROTOCOLS) * 5 + ["iec61850", None, ["iec104"]]),
                        ("listen", ["this_pc", "network"] * 6 + ["all", None, ["this_pc"], {"a": 1}]),
                        ("role", ["gateway", "master", "converter"] * 5 + ["station", None, ["gateway"]]),
                        ("real_commands", [True, False, 1, None]), ("auto_start", [True, False, "yes"])):
        if r.random() < 0.6:
            raw[key] = r.choice(values)
    if r.random() < 0.5:
        raw["master"] = mx.random_master(r) if r.random() < 0.5 else mx.valid_master(r)
    if r.random() < 0.6:
        raw["settings"] = mx.per_protocol(r, mx.random_settings) if r.random() < 0.4 else {
            "iec104": {"t1": 20, "t2": r.randint(5, 25)}, "dnp3": {"unsolicited": r.choice([True, False, 1])},
            "modbus": {"unit_id": r.randint(0, 260)}}
    if r.random() < 0.7:
        raw["points"] = mx.per_protocol(r, lambda _: [stored_point() for _ in range(r.randint(0, 5))]
                                        if r.random() < 0.9 else r.choice(fx.JUNK))
    return raw


for _ in range(count):
    protocol, values = r.choice(PROTOCOL_ARGS), mx.random_settings(r)
    fx.case("validate_settings", [protocol, values], lambda: gr.validate_settings(protocol, fx.fresh(values)))
    raw = random_gateway()
    fx.case("normalize_gateway", [raw], lambda: gr.normalize_gateway(fx.fresh(raw)))
    choice = r.choice(fx.JUNK + ["this_pc", "network", "This_pc"])
    fx.case("listen_address", [choice], lambda: gr.listen_address(fx.fresh(choice)))
for kind in fx.VALUE_KINDS + ["", "INT", "FLOATING", "Enumeration", "boolean", "Dbpos2"]:
    fx.case("allowed_kinds", [kind], lambda: gr.allowed_kinds(kind))
fx.case("empty_gateway", [], gr.empty_gateway)
fx.case("normalize_gateway", [{"points": {"iec104": [{"kind": "single", "address": 1}] * 5001}}],
        lambda: gr.normalize_gateway({"points": {"iec104": [{"kind": "single", "address": 1}] * 5001}}))

sets = {f"set{number}": fx.ied_set(r, r.randint(1, 5)) for number in range(int(count / 8) + 2)}
for number, ieds in enumerate(fx.broken_sets()):
    sets[f"broken{number}"] = ieds
for name, ieds in sets.items():
    fx.emit_fixture(name, ieds)
    for _ in range(1 if name.startswith("broken") else 5):
        protocol = r.choice(gp.PROTOCOLS)
        points = (random_points(ieds, protocol) if not name.startswith("broken") else
                  [{"kind": r.choice(["double", "command_double"]), "ied": "b1", "ref": "LD0/XCBR1.Pos.stVal",
                    "address": 7}])
        fx.case("validate_points", [name, protocol, points], lambda: gr.validate_points(protocol, fx.fresh(points),
                                                                                         fx.fresh(ieds)))
big = fx.big_set(r, 3, 2200)
fx.emit_fixture("big", big)
proposal = gp.propose(fx.fresh(big), "iec104")[0]
fx.case("validate_points", ["big", "iec104", proposal], lambda: gr.validate_points("iec104", fx.fresh(proposal), big))
fx.case("validate_points", ["big", "dnp3", proposal + [{}]], lambda: gr.validate_points("dnp3", proposal + [{}], big))
fx.case("validate_points", ["big", "modbus", proposal], lambda: gr.validate_points("modbus", fx.fresh(proposal), big))
