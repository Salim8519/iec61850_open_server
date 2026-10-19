"""Goldens for the SLD import and signal-flow domain (scl/substation, domain/sld_*,
scl/flow_sources, domain/signal_flow, projects/signal_flow.edge_csv).

  python studio/engine/internal/scl/testdata/sld_flow_gen.py   -> golden/sld_flow.json

Inputs: every public fixture and UTF-8 corpus case, the unit-test SLD fixture builder
(tests/unit/sld_fixtures.py) with seeded random stations, and open_substation variants.
Per input: {name, case | text, substation, diagram, bindings, graphs, csv}, each part a
result or {"error": {...}}. Bindings use IEDs built from the LNodes (deterministic).
"""
import json
import random
import pathsetup

pathsetup.install()

import sld_fixtures as fx  # noqa: E402  (tests/unit, public)
from golden_common import capture  # noqa: E402
from golden_gen import FIXTURES, CORPUS_PREFIX  # noqa: E402
from studio_app.domain.signal_flow import build_graph  # noqa: E402
from studio_app.domain.sld_bindings import bind  # noqa: E402
from studio_app.domain.sld_layout import build_diagram  # noqa: E402
from studio_app.projects import signal_flow  # noqa: E402
from studio_app.scl.document import parse_document  # noqa: E402
from studio_app.scl.flow_sources import read_flow  # noqa: E402
from studio_app.scl.substation import read_substation  # noqa: E402

KINDS = ["CBR", "DIS", "CTR", "VTR", "IFL", "CAP"]


def random_station(seed):
    """Seeded stations: busbars, feeders, tees, grounded switches, missing nodes, LNodes."""
    rng = random.Random(seed)
    bays, esld = [("BB", "", ["L", "M"])], rng.random() < 0.5
    for bay in range(rng.randint(1, 4)):
        nodes = [f"N{k}" for k in range(rng.randint(1, 4))]
        items = ""
        for k in range(rng.randint(1, 6)):
            pool = [f"S/V/F{bay}/{n}" for n in nodes] + ["S/V/BB/L", "S/V/BB/M", "grounded", "S/V/X/missing"]
            ends = rng.sample(pool, rng.randint(1, 3))
            lnodes = ""
            if rng.random() < 0.6:
                ied = rng.choice(["IEDA", "IEDB", "None", ""])
                for cls in rng.sample(["XCBR", "XSWI", "CSWI", "PTOC"], 2):
                    lnodes += (f'<LNode iedName="{ied}" ldInst="{rng.choice(["LD0", "LD9"])}" lnClass="{cls}" '
                               f'lnInst="{rng.randint(1, 2)}" prefix="{rng.choice(["", "Q"])}"/>')
            at = (rng.randint(0, 6), rng.randint(0, 6) + rng.choice([0, 0.5])) if esld and rng.random() < 0.9 else None
            items += fx.equipment(f"E{k}", rng.choice(KINDS), ends, lnodes, at)
        bays.append((f"F{bay}", items, nodes))
    return fx.scl(bays, extra='<PowerTransformer name="T1" type="PTR"/>' if seed % 3 == 0 else "")


def fake_ieds(topology):
    """IEDs named by the LNodes: Pos.stVal signals and Pos controls with varied kinds."""
    names = sorted({lnode["iedName"] for item in topology["equipment"] for lnode in item["lnodes"]})
    ieds = []
    for index, name in enumerate(names):
        signals, controls = [], []
        for item in topology["equipment"]:
            for n, lnode in enumerate(item["lnodes"]):
                domain = topology["ld_domains"].get((lnode["iedName"], lnode["ldInst"]))
                if lnode["iedName"] != name or domain is None:
                    continue
                ref = f"{domain}/{lnode['prefix']}{lnode['lnClass']}{lnode['lnInst']}.Pos"
                signals.append({"ref": ref + ".stVal", "kind": "Dbpos" if n % 3 else "BOOLEAN",
                                "writable": n % 4 != 3, "fc": "ST"})
                controls.append({"ref": ref, "cdc": "DPC" if n % 2 == 0 else "SPC"})
        ieds.append({"id": f"ied{index}", "name": name, "signals": signals, "controls": controls})
    return ieds


def jsonable_topology(topology):
    return {**topology, "ld_domains": [[list(key), value] for key, value in topology["ld_domains"].items()]}


class _Service:
    def __init__(self, text, names):
        import threading
        self.lock = threading.RLock()
        self.project = {"ieds": [{"id": f"i{n}", "name": name, "filename": "f.scd", "source": text}
                                 for n, name in enumerate(names)]}


def flow_part(text):
    ok, root = capture(parse_document, text)
    names = [ied.get("name") for ied in root.findall("IED")] if ok else []
    graphs = []
    for ied in [None] + names[:2] + ["NOPE"]:
        ok, flow = capture(read_flow, text)
        result = capture(build_graph, flow, set(names[:1]), ied) if ok else (False, flow)
        graphs.append({"ied": ied, "result" if result[0] else "error": result[1]})
    csv = []
    if names:
        service = _Service(text, names[:1])
        source = signal_flow.flow_sources(service)[0]["id"]
        ok, full = capture(signal_flow.graph, service, source)
        for edge_id in ([edge["id"] for edge in full["edges"]][:3] if ok else []) + ["nope"]:
            ok, value = capture(signal_flow.edge_csv, service, source, edge_id)
            csv.append({"edge": edge_id, "loaded": names[:1], "result" if ok else "error": value})
    return graphs, csv


def entry(name, text, reference):
    item = {"name": name, **reference}
    ok, topology = capture(read_substation, text)
    item["substation"] = jsonable_topology(topology) if ok else {"error": topology}
    if ok:
        built_ok, built = capture(build_diagram, topology, "src")
        item["diagram"] = built if built_ok else {"error": built}
        ieds = fake_ieds(topology)
        item["ieds"] = ieds
        item["bindings"] = [list(bind(node["data"]["kind"], built["equipment"][node["id"]]["lnodes"], ieds,
                                      topology["ld_domains"]))
                            for node in built["diagram"]["nodes"] if node["id"] in built["equipment"]] if built_ok else []
    item["graphs"], item["csv"] = flow_part(text)
    return item


def main():
    entries = []
    cases = list(FIXTURES)
    manifest = json.loads((pathsetup.CORPUS / "manifest.json").read_bytes())
    cases += [(item["case"], CORPUS_PREFIX + item["case"] + ".scd") for item in manifest]
    for case, path in cases:
        try:
            text = (pathsetup.REPO / path).read_bytes().decode("utf-8-sig")
        except UnicodeDecodeError:
            continue
        entries.append(entry(case, text, {"case": path}))
    for seed in range(120):
        text = random_station(seed)
        entries.append(entry(f"random {seed}", text, {"text": text}))
    for name, text in [("feeder", fx.feeder()), ("feeder esld", fx.feeder(True))]:
        entries.append(entry(name, text, {"text": text}))
    scd = (pathsetup.REPO / "scd" / "open_substation.scd").read_text(encoding="utf-8")
    variants = {"named": scd.replace('srcCBName="GoCB"', 'srcCBName="gcbEvents"'),
                "by data": scd.replace(' srcLDInst="GenericIO" srcLNClass="LLN0" srcCBName="GoCB"', ""),
                "clients": scd.replace("<RptEnabled", '<RptEnabled max="2"><ClientLN iedName="IED2_PTOC" '
                                       'ldInst="GenericIO" lnClass="IHMI" lnInst="1"/><ClientLN iedName="SCADA1" '
                                       'ldInst="LD0" lnClass="ITCI" lnInst="1"/></RptEnabled><RptEnabled', 1)}
    for name, text in variants.items():
        entries.append(entry("open_substation " + name, text, {"text": text}))
    pathsetup.write_bytes(pathsetup.GOLDEN / "sld_flow.json", json.dumps(entries).encode("ascii"))
    print(len(entries), "entries")


if __name__ == "__main__":
    main()
