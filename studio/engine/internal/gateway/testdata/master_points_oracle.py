"""Oracle for master_points: merge, quality words, value text, rows, CSV export and labels."""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import gateway_fixtures as fx  # noqa: E402
import master_values as mv  # noqa: E402
from oracle_lib import rng, seed  # noqa: E402
from studio_app.domain import master_points as mp  # noqa: E402

r = rng(seed())
count = int(sys.argv[2]) if len(sys.argv) > 2 else 1500
PROTOCOLS = ["iec104", "dnp3", "modbus"] * 3 + ["other"]
LABEL_HEADERS = ["group,address,label", "Group , ADDRESS,Label,extra", "group,address", "label,address,group,group",
                 "group,address,label,label", "", "\n", "address,label,GROUP", "group;address;label",
                 "K,group,address,label", "group,,address,label", '"group","address","label"']


def label_text():
    lines = [r.choice(LABEL_HEADERS)]
    for _ in range(r.randint(0, 10)):
        cells = [r.choice(mv.GROUP_NAMES + ["bogus", "", " single "]),
                 r.choice(["1", "007", "", "x", "1.5", "١٢", "²", "-1", "9" * 30, "9" * 4301,
                           " 12 ", "①"]),
                 r.choice(["Breaker", "", " ", "x" * 100, "a,b", 'q"q', "\U0001F600"])]
        choice = r.random()
        if choice < 0.15:
            cells = cells[:r.randint(0, 2)]
        elif choice < 0.25:
            cells += ["extra"]
        line = ",".join('"' + cell.replace('"', '""') + '"' if any(c in cell for c in ',"') else cell
                        for cell in cells)
        lines.append(line if r.random() < 0.9 else r.choice(["", ",,", "\r"]))
    text = r.choice(["\n", "\r\n"]).join(lines)
    return text + r.choice(["", "\n", '"', "\x00"]) if r.random() < 0.3 else text


for _ in range(count):
    batches = mv.merge_batches(r)
    steps, table = mv.merge_sequence(batches)
    fx.case("merge_sequence", [batches], lambda: steps)
    protocol = r.choice(PROTOCOLS)
    unticked = r.sample(sorted(table) + ["single:999"], min(len(table), r.randint(0, 3)))
    labels = {key: r.choice(["Breaker", "", "x" * 10]) for key in r.sample(sorted(table), min(len(table), 3))}
    link_open = r.random() < 0.6
    args = [table, protocol, unticked, labels, link_open]
    fx.case("rows", args, lambda: mp.rows(*fx.fresh(args)))
    try:
        display = mp.rows(*fx.fresh(args))
    except Exception:  # noqa: BLE001  (the rows case above recorded the error)
        display = None
    if display is not None:
        fx.case("export_csv", [display], lambda: mp.export_csv(fx.fresh(display)))
    quality = r.choice(mv.QUALITIES + [r.randint(-300, 300), r.randint(0, 2 ** 80), 0, None, 1.5, "x"])
    group = r.choice(mv.GROUP_NAMES + [None])
    if isinstance(quality, int) or protocol not in ("iec104", "dnp3"):
        fx.case("quality_words", [protocol, quality, group], lambda: mp.quality_words(protocol, quality, group))
    row = {"group": r.choice(mv.GROUP_NAMES)}
    if r.random() < 0.9:
        row["value"] = mv.master_value(r, 0).get("value") if r.random() < 0.8 else r.choice([[1], {"a": 1}])
    fx.case("value_text", [row], lambda: mp.value_text(fx.fresh(row)))
    value = {"group": r.choice(mv.GROUP_NAMES), "address": r.choice([1, True, 2 ** 70, -1])}
    fx.case("key", [value], lambda: mp.key(fx.fresh(value)))
    text = label_text()
    fx.case("import_labels", [text], lambda: mp.import_labels(text))
for protocol in PROTOCOLS[-4:]:
    for group in mv.GROUP_NAMES:
        fx.case("group_name", [protocol, group], lambda: mp.group_name(protocol, group))
fx.case("value_text", [{"value": 1}], lambda: mp.value_text({"value": 1}))
for text in ["", "group,address,label\n" + "single,1," + "x" * 131073, "group,address,label\nsingle,²,L\n",
             "group,address,label\nsingle," + "9" * 4301 + ",L\n", 'group,address,label\n"single', "\n\ngroup",
             "group,address,label\n\n\nsingle,1,A\n,,\nbogus,1,B\ncoil,2,C"]:
    fx.case("import_labels", [text], lambda: mp.import_labels(text))
