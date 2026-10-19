"""IED Explorer domain rules on random synthetic browse answers and read values."""
import os
import sys

sys.path.insert(0, os.environ["STUDIO_ORACLE_LIB"])
from oracle_lib import case, rng, seed  # noqa: E402
from studio_app.domain.explorer_tree import node_view  # noqa: E402
from studio_app.domain.explorer_values import describe, time_text  # noqa: E402
from studio_app.domain.explorer_watch import key_nodes, key_points, merge  # noqa: E402

random = rng(seed())
CLASSES = ["XCBR", "XSWI", "CSWI", "MMXU", "MMXN", "MSQI", "PTRC", "LLN0", "LPHD", "GGIO", "PDIS", "MHAI"]
FCS = ["ST", "MX", "CO", "SP", "CF", "DC", "EX", "SV", "BR", "RP", "GO", "XX", "OR", "BL", "SG", "SE"]
OBJECTS = ["Pos", "Beh", "Health", "PhyHealth", "Tr", "Op", "Str", "TotW", "A", "PhV", "Mod", "NamPlt"]
LEAVES = ["stVal", "q", "t", "general", "mag.f", "phsA.cVal.mag.f", "ctlModel", "d", "Oper.ctlVal", "instMag.i"]


def node_name():
    return random.choice(["", "Q1", "QA1", "AB"]) + random.choice(CLASSES) + random.choice(["", "1", "2", "12"])


def answer(reference):
    leaves = []
    for _ in range(random.randint(0, 30)):
        obj = random.choice(OBJECTS)
        leaf = obj if random.random() < 0.05 else f"{obj}.{random.choice(LEAVES)}"
        leaves.append({"ref": f"{reference}.{leaf}", "fc": random.choice(FCS),
                       "mms_type": random.choice([0, 1, 2, 3, 4, 6, 7, 15, 16, 99, 2.0, True])})
    data = {"leaves": leaves, "datasets": [f"DS{n}" for n in range(random.randint(0, 3))],
            "dataset_members": {"DS0": {"members": ["a$b"]}, "DS1": {}, "DS2": None},
            "buffered_reports": [f"brcb{n}" for n in range(random.randint(0, 2))],
            "unbuffered_reports": [f"urcb{n}" for n in range(random.randint(0, 2))]}
    if random.random() < 0.5:
        data["name"] = reference.split("/")[-1] + "x"
    if random.random() < 0.1:
        del data["leaves"]
    return data


def raw_value():
    return random.choice([
        True, False, None, 0, 7, -3, 1.5, 230.123456789, 1e-7, 2 ** 40, "text", "", [1, 2], {"access_error": 3},
        {"bits": 2, "be": random.randint(0, 3), "le": random.randint(0, 3)},
        {"bits": 13, "be": random.randint(0, 8191), "le": random.randint(0, 8191)},
        {"bits": random.randint(0, 9), "be": random.randint(-5, 300), "le": 0}, {"octets": ""}, {"octets": "0a0b"},
        {"other": 1}, random.randint(1, 4102444800000), 0, -1])


for number in range(int(sys.argv[2]) if len(sys.argv) > 2 else 1500):
    reference = random.choice(["LD0", "IED1CTRL", "éLD"]) + "/" + node_name()
    data = answer(reference)
    case("node_view", [reference, data], lambda: node_view(reference, data))
    try:
        view = node_view(reference, data)
    except Exception:
        view = None
    if view is not None:
        case("key_points", [view], lambda: key_points(view))
    names = [node_name() for _ in range(random.randint(0, 25))]
    case("key_nodes", ["LD0", names], lambda: key_nodes("LD0", names))
    points = [{"ref": random.choice(["a", "b", "c"]) + str(random.randint(0, 30)), "fc": random.choice(["ST", "MX"]),
               "type": "x"} for _ in range(random.randint(0, 80))]
    case("merge", [points], lambda: merge(points))
    attribute = random.choice(["q", "stVal", "t", "T", "mag.f", "general"])
    raw = raw_value()
    ref = f"LD0/XCBR1.Pos.{attribute}"
    case("describe", [ref, raw], lambda: describe(ref, raw))
    case("time_text", [raw], lambda: time_text(raw))
