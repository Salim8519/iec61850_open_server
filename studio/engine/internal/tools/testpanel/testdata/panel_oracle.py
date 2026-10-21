"""validate_panel on template IEDs (public) with random saved controls."""
import os
import sys

sys.path.insert(0, os.environ["STUDIO_ORACLE_LIB"])
from oracle_lib import case, rng, seed  # noqa: E402
from studio_app.projects.test_panel import validate_panel  # noqa: E402
from studio_app.scl.importer import import_scl  # noqa: E402
from studio_app.scl.templates import make_template  # noqa: E402

random = rng(seed())
ieds = []
for number, name in enumerate(("BayA", "BayB")):
    ied = import_scl(make_template(name))[0]
    ied["id"] = "ied%d" % number
    ieds.append(ied)
refs = [(ied["id"], signal["ref"]) for ied in ieds for signal in ied["signals"]]
reports = [report["ref"] for ied in ieds for report in ied["reports"]] + ["nope", 5, None, ""]
print(__import__("json").dumps({"ieds": ieds}))

for number in range(int(sys.argv[2]) if len(sys.argv) > 2 else 1500):
    entries = []
    for index in range(random.randint(0, 4)):
        ied_id, ref = random.choice(refs)
        entry = {"id": random.choice(["c%d" % index, "c0", "bad id", "x" * 65, 5]),
                 "label": random.choice(["Trip", "  ", "x" * 101, " ok ", 3]),
                 "ied_id": random.choice([ied_id, ied_id, "missing", None]), "ref": random.choice([ref, ref, "LD0/X.y", 1])}
        for key, choices in (("report_ref", reports), ("duration_ms", [1000, 99, 60000, True, 1.5, "100"]),
                             ("active_value", [True, False, 1])):
            if random.random() < 0.5:
                entry[key] = random.choice(choices)
        entries.append(entry if random.random() < 0.95 else random.choice([None, [1], "x"]))
    if random.random() < 0.02:
        entries = random.choice([None, {"a": 1}, [{}] * 501])
    case("validate_panel", [entries], lambda: validate_panel(entries, ieds))
