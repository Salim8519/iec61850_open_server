"""Oracle for gateway_csv: point lists exported and imported as CSV."""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import gateway_fixtures as fx  # noqa: E402
from oracle_lib import rng, seed  # noqa: E402
from studio_app.domain import gateway_csv as gc  # noqa: E402
from studio_app.domain import gateway_points as gp  # noqa: E402
from studio_app.domain import gateway_rules as gr  # noqa: E402

r = rng(seed())
count = int(sys.argv[2]) if len(sys.argv) > 2 else 2000
HEADERS = ["type,address,ied,reference,scale,description", "TYPE, Address ,IED,Reference",
           "﻿type,address,ied,reference", "﻿﻿type,address,ied,reference,scale",
           "type,address,ied", "type;address;ied;reference", "Type,Address,İed,Reference",
           "type,address,ied,reference,x,y,z", '"type","address","ied","reference"', "", "﻿"]
ADDRESSES = ["1", "0", "-3", "x", "1.5", "1_000", " 12", "١٢", "9" * 4301, "0" * 4300 + "7", "",
             "16777215", "99999999999999999999", "+5", "1__0"]
SCALES = ["", "1", "2.5", "1e3", "inf", "-nan", "x", "1_0.5", "0x10", " 3 ", "1e999", "٣.5", "."]


def odd_point(point):
    point = dict(point)
    choice = r.random()
    if choice < 0.15:
        point["scale"] = r.choice([2, True, 10 ** 20, 1e-7, -0.0, float("nan"), float("inf"), 123456789.0])
    elif choice < 0.2:
        point["scale"] = r.choice(["x", None, [1], 10 ** 400])
    elif choice < 0.25:
        point.pop("scale")
    elif choice < 0.3:
        point["address"] = r.choice([True, False, 2 ** 70, -5])
    elif choice < 0.35:
        point["description"] = r.choice(fx.TEXTS + [None, 5])
    elif choice < 0.4:
        point["ied"] = r.choice(["unknown", 5, None])
    elif choice < 0.42:
        point.pop(r.choice(["ied", "ref", "description"]))
    return point


def random_row(ieds):
    kind = r.choice(list(gp.KINDS) * 3 + ["analog", "", " single "])
    name = r.choice([ied["name"] for ied in ieds] + ["Nobody", "", " " + ieds[0]["name"]])
    cells = [kind, r.choice(ADDRESSES), name, fx.random_ref(r), r.choice(SCALES), r.choice(fx.TEXTS)]
    choice = r.random()
    if choice < 0.1:
        cells = cells[:r.randint(0, 5)]
    elif choice < 0.15:
        cells += ["extra", "more"]
    elif choice < 0.2:
        cells = [" "] * r.randint(1, 6)
    return ",".join('"' + cell.replace('"', '""') + '"' if any(c in cell for c in ',"\n') else cell
                    for cell in cells)


def random_text(ieds, exported):
    lines = exported.split("\r\n")[:-1] if exported and r.random() < 0.5 else [r.choice(HEADERS)]
    if r.random() < 0.2:
        lines[0] = r.choice(HEADERS)
    for _ in range(r.randint(0, 12)):
        lines.insert(r.randint(1, len(lines)), random_row(ieds) if r.random() < 0.8 else r.choice(["", ",,,", "\r"]))
    text = r.choice(["\r\n", "\n"]).join(lines)
    return text + r.choice(["", "\n", "\r\n", '"', "\x00"]) if r.random() < 0.3 else text


sets = {f"set{number}": fx.ied_set(r, r.randint(1, 4)) for number in range(int(count / 20) + 2)}
for name, ieds in sets.items():
    fx.emit_fixture(name, ieds)
    for _ in range(10):
        protocol = r.choice(gp.PROTOCOLS)
        proposal = gp.propose(fx.fresh(ieds), protocol)[0]
        clean = gr.validate_points(protocol, r.sample(proposal, min(len(proposal), r.randint(0, 30))), ieds)[0]
        points = [odd_point(point) if r.random() < 0.3 else point for point in clean]
        fx.case("export_csv", [name, points], lambda: gc.export_csv(fx.fresh(points), fx.fresh(ieds)))
        exported = gc.export_csv(clean, ieds)
        for _ in range(2):
            text = random_text(ieds, exported)
            fx.case("import_csv", [name, text], lambda: gc.import_csv(text, fx.fresh(ieds)))
broken = [{"id": "b1", "name": "B1"}, {"id": "b2"}]
fx.emit_fixture("broken", broken)
fx.case("import_csv", ["broken", "type,address,ied,reference\n"], lambda: gc.import_csv("type,address,ied,reference\n", broken))
fx.case("export_csv", ["broken", []], lambda: gc.export_csv([], broken))
fx.emit_fixture("odd", [{"id": ["x"], "name": "B1"}])
fx.case("export_csv", ["odd", []], lambda: gc.export_csv([], [{"id": ["x"], "name": "B1"}]))
huge_field = "type,address,ied,reference\nsingle,1,Bay1," + "r" * 131073 + "\n"
fx.case("import_csv", ["set0", huge_field], lambda: gc.import_csv(huge_field, fx.fresh(sets["set0"])))
too_big = "type,address,ied,reference\n" + "x" * (2 * 1024 * 1024)
fx.case("import_csv", ["set0", too_big], lambda: gc.import_csv(too_big, fx.fresh(sets["set0"])))
