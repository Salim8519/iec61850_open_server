"""Synthetic gateway oracle inputs: public-template IEDs plus random mutations.

Only studio_app.scl.templates (the documented lab template) is used; never test_cids or
customer files. Every value is JSON-only, so Go decodes exactly what Python receives.
"""
import json
import os
import sys

sys.path.insert(0, os.environ["STUDIO_ORACLE_LIB"])
sys.path.insert(0, os.environ["STUDIO_BACKEND"])
from oracle_lib import case  # noqa: E402,F401  (re-exported for the scripts)
from studio_app.scl.importer import import_scl  # noqa: E402
from studio_app.scl.templates import make_template  # noqa: E402

LNS = ["XCBR1", "XSWI2", "CSWI1", "CILO1", "GGIO1", "GGIO12", "SIMG1", "SIML1", "SARC1", "SPDC1", "SCBR1",
       "SOPM1", "MMXU1", "MMXN1", "MSQI1", "MHAI1", "MMTR1", "MSTA1", "ATCC1", "YLTC1", "PTOC1", "PDIS2",
       "RREC1", "RBRF1", "LPHD1", "LLN0", "QA1XCBR1", "Q0CSWI1", "xcbr1", "XCB1", "ABCDE", "GGIO١",
       "PTOC1\n", "TCTR1", "MMXU", "LLN0X", "GGIO1\n\n", "Q1XSWI\U0001D7D9"]
DOS = ["Pos", "Ind1", "Ind2", "SPCSO1", "Alm", "Op", "Str", "Health", "Beh", "PhyHealth", "TotW", "A", "PhV",
       "Hz", "ThdA", "ThdPhV", "HA", "AnIn1", "TapChg", "Mod", ""]
ATTRS = ["stVal", "general", "mag.f", "mag.i", "q", "t", "phsA.cVal.mag.f", "ctlVal", "instMag.i"]
FCS = ["ST", "MX", "CO", "CF", "DC"]
VALUE_KINDS = ["Dbpos", "BOOLEAN", "FLOAT32", "FLOAT64", "INT32", "INT8U", "INT32U", "Enum", "Quality",
               "Timestamp", "VisString255", "INT64"]
JUNK = [None, True, False, 0, 1, -1, 7, 2 ** 63, -2 ** 63 - 1, 10 ** 30, 1.5, 0.0, -0.0, float("nan"),
        float("inf"), "", "x", " 5 ", "５", [], [1], {}, {"a": 1}]
TEXTS = ["", "Breaker", " ", "Feeder é", "x" * 130, "\U0001F600" * 3, "a,b", 'say "hi"', "line\nbreak"]


def fresh(value):
    """What Go decodes: a JSON round trip (lists instead of tuples, no shared objects)."""
    return json.loads(json.dumps(value))


def random_ref(r):
    ld = r.choice(["LD0", "CTRL", "PROT", "", "A/B"])
    ref = f"{ld}/{r.choice(LNS)}.{r.choice(DOS)}.{r.choice(ATTRS)}"
    return ref if r.random() < 0.95 else ref.split("/", 1)[-1]


def random_signal(r):
    signal = {"ref": random_ref(r), "fc": r.choice(FCS), "kind": r.choice(VALUE_KINDS)}
    if r.random() < 0.7:
        signal["description"] = r.choice(["", "", "Spare", None, 0, "Trip"])
    return signal


_TEMPLATES = {}


def template_ied(r, number):
    name = f"Bay{number}"
    if name not in _TEMPLATES:
        _TEMPLATES[name] = import_scl(make_template(name))[0]
    ied = _TEMPLATES[name]
    result = {"id": f"ied{number}", "name": name, "state": ied["state"],
              "signals": [{key: signal[key] for key in ("ref", "fc", "kind", "description")}
                          for signal in ied["signals"]],
              "controls": [{key: control[key] for key in ("ref", "status", "cdc")} for control in ied["controls"]],
              "datasets": [{"name": item["name"], "signal_refs": list(item["signal_refs"])} for item in ied["datasets"]]}
    role = r.choice([None, "simulated", "real", "off", "monitor"])
    if role:
        result["role"] = role
    return result


def mutate_ied(r, ied):
    for _ in range(r.randint(0, 40)):
        ied["signals"].append(random_signal(r))
    for _ in range(r.randint(0, 3)):
        control = {"ref": random_ref(r).rsplit(".", 1)[0]}
        if r.random() < 0.8:
            control["cdc"] = r.choice(["DPC", "SPC", "INC", None])
        ied["controls"].append(control)
    choice = r.random()
    if choice < 0.15:
        ied.pop("datasets")
    elif choice < 0.3:
        ied["datasets"] = []
    elif choice < 0.6:
        refs = [signal["ref"] for signal in r.sample(ied["signals"], min(8, len(ied["signals"])))]
        ied["datasets"].append({"name": "Extra", "signal_refs": refs})
    elif choice < 0.65:
        ied["datasets"].append({"name": "NoRefs"})
    if r.random() < 0.1:
        ied.pop("controls")
    r.shuffle(ied["signals"])
    return ied


def ied_set(r, size):
    ieds = [mutate_ied(r, template_ied(r, r.randint(1, 30))) for _ in range(size)]
    if size > 1 and r.random() < 0.2:
        ieds[-1]["name"] = ieds[0]["name"]  # duplicate names: the later IED wins a name lookup
    return ieds


def big_set(r, count, per_ied):
    """IEDs offering more than MAX_POINTS candidates (GGIO spares described, measurements)."""
    ieds = []
    for number in range(count):
        ied = template_ied(r, 100 + number)
        for index in range(per_ied):
            ln = r.choice(["GGIO7", "MMXU2", "XCBR3", "PTOC4"])
            kind = {"GGIO7": "BOOLEAN", "MMXU2": "FLOAT32", "XCBR3": "Dbpos", "PTOC4": "BOOLEAN"}[ln]
            attr = {"GGIO7": "Ind%d.stVal", "MMXU2": "A%d.mag.f", "XCBR3": "Pos%d.stVal", "PTOC4": "Op%d.general"}[ln]
            fc = "MX" if ln == "MMXU2" else "ST"
            ied["signals"].append({"ref": f"LD0/{ln}.{attr % index}", "fc": fc, "kind": kind,
                                   "description": r.choice(["", "Spare input"])})
        ieds.append(ied)
    return ieds


def broken_sets():
    """IED models that make Python crash, for the plain-error paths."""
    good = {"id": "b1", "name": "B1", "signals": [{"ref": "LD0/XCBR1.Pos.stVal", "fc": "ST", "kind": "Dbpos"}],
            "controls": [{"ref": "LD0/XCBR1.Pos", "cdc": "DPC"}]}
    cases = []
    for key in ("signals", "id", "name"):
        cases.append([{k: v for k, v in good.items() if k != key}])
    cases.append([dict(good, controls=None)])
    cases.append([dict(good, datasets=None)])
    cases.append([dict(good, datasets=[5])])
    cases.append([dict(good, datasets=[{"signal_refs": [[1]]}])])
    cases.append([dict(good, datasets=[{"signal_refs": "LD0/XCBR1.Pos.stVal"}])])
    cases.append([dict(good, datasets=[{"signal_refs": [5]}], signals=[{"ref": "LD0/GGIO1.Ind1.stVal", "fc": "ST", "kind": "BOOLEAN"}])])
    cases.append([dict(good, signals=[{"ref": 5, "fc": "ST", "kind": "Dbpos"}])])
    cases.append([dict(good, signals=[{"ref": "LD0/LLN0.Health.stVal", "fc": "ST", "kind": 5}])])
    cases.append([dict(good, signals=[{"ref": "LD0/MMXU1.A.mag.f", "fc": "MX", "kind": None}])])
    cases.append([dict(good, signals=[{"ref": "LD0/XCBR1.Pos.stVal", "fc": "ST"}])])
    cases.append([dict(good, signals=[["x"]])])
    cases.append([dict(good, signals={"a": 1})])
    cases.append([dict(good, signals=None)])
    cases.append([dict(good, controls=[{"cdc": "DPC"}])])
    cases.append([dict(good, controls=[{"ref": 7, "cdc": "DPC"}])])
    cases.append([dict(good, controls=[{"ref": [7]}])])
    cases.append([dict(good, controls=["x"])])
    cases.append([dict(good, id=["unhashable"])])
    cases.append([dict(good, name=["unhashable"])])
    cases.append([dict(good, signals=[{"ref": "LD0/MMXU1.A.mag.f", "fc": "MX", "kind": 5}])])
    return cases


def emit_fixture(name, ieds):
    print(json.dumps({"fn": "fixture", "args": [name], "result": ieds}))
