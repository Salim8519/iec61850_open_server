"""Oracle for gateway_selection and gateway_points (node_class, smart_kind, propose ...)."""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import gateway_fixtures as fx  # noqa: E402
from oracle_lib import rng, seed  # noqa: E402
from studio_app.domain import gateway_points as gp  # noqa: E402
from studio_app.domain import gateway_selection as gs  # noqa: E402

r = rng(seed())
count = int(sys.argv[2]) if len(sys.argv) > 2 else 2000

for _ in range(count):
    ref = fx.random_ref(r)
    fx.case("node_class", [ref], lambda: gs.node_class(ref))
    signal = fx.random_signal(r)
    fx.case("smart_kind", [signal], lambda: gs.smart_kind(fx.fresh(signal)))
    refs = r.choice([None, [], [signal["ref"]], [fx.random_ref(r) for _ in range(3)]])
    fx.case("engineered", [signal, refs], lambda: gs.engineered(fx.fresh(signal), None if refs is None else set(refs)))
    kind = r.choice(gp.KINDS)
    protocol = r.choice(gp.PROTOCOLS)
    point = {"kind": kind, "address": r.randint(0, 70000)}
    fx.case("occupied", [protocol, point], lambda: gp.occupied(protocol, fx.fresh(point)))
for text in ["LD0/LLN0.Mod.stVal", "LLN0", "", "/", "a/b/c.d", "LD0/XCBR١٢.Pos", "LD0/XCBR1\n.Pos",
             "LD0/xXCBR1", "LD0/XCBR1\n\n", "LD0/ABC1", "LD0/ABCD", "LD0/ABCDE12", "LD0/\U0001D7D9XCBR1"]:
    fx.case("node_class", [text], lambda: gs.node_class(text))
fx.case("point_id", ["ied1", "LD0/XCBR1.Pos.stVal", "double"], lambda: gp.point_id("ied1", "LD0/XCBR1.Pos.stVal", "double"))

sets = {}
for number in range(int(count / 25) + 4):
    sets[f"set{number}"] = fx.ied_set(r, r.randint(0, 6))
sets["big"] = fx.big_set(r, 3, 2200)
sets["big_off"] = fx.big_set(r, 2, 2600)
sets["big_off"][0]["role"] = "off"
for number, ieds in enumerate(fx.broken_sets()):
    sets[f"broken{number}"] = ieds
for name, ieds in sets.items():
    fx.emit_fixture(name, ieds)
    for index, ied in enumerate([] if name.startswith("broken") else ieds[:2]):
        refs = gs.dataset_refs(fx.fresh(ied))
        if refs is None or all(isinstance(ref, str) for ref in refs):
            fx.case("dataset_refs", [name, index], lambda: refs)
    for protocol in gp.PROTOCOLS + ("iec61850",):
        if name.startswith("big") and protocol != "modbus" and r.random() < 0.5:
            continue
        fx.case("propose", [name, protocol], lambda: gp.propose(fx.fresh(ieds), protocol))
