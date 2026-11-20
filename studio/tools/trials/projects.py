"""Build the trial projects in an instance and put every IED on the ScadaBench Lab card.

The two Step 0 projects (local timing only; the data never leaves the PC):
- `small`: about 7 IEDs, the 8-IED panels project in test_cids/substation (diagram included).
- `large`: about 20 IEDs, the 14-IED 132 kV project in test_cids/Al Kamil plus the first six
  differently named IEDs of the 400 kV project (their SCL sources travel with them).
Every IED gets MMS port 20102 (ports >= 20000) and a Lab-card address 198.18.20.<10+n>
through POST /network/lab/use, the same call the Devices page makes. The Lab card refuses
loopback (127.x) addresses, so trials use the 198.18.0.0/15 test range; the lead gave this
harness 198.18.20.0/24 (other trials use other /24s of that range).

    document = projects.build("small")
    projects.open_in(engine, document)
    addresses = projects.use_lab_card(engine)   # {ied_id: ip}
"""
import copy
import io
import json
from pathlib import Path
import zipfile

REPO = Path(__file__).resolve().parents[3]
CIDS = REPO / "test_cids"
SOURCES = {
    "small": [(CIDS / "substation/DUQM-panels.scadabench", None)],
    "large": [(CIDS / "Al Kamil/AlKamil-132kV.scadabench", None),
              (CIDS / "Al Kamil/AlKamil-400kV.scadabench", 6)],
}
IED_PORT = 20102
RANGE = "198.18.20"  # this harness's /24 of the 198.18.0.0/15 test range


def read(path):
    with zipfile.ZipFile(path) as archive:
        return json.loads(archive.read("project.json"))


def build(name):
    """The project document for `small` or `large` (first file whole, later files' IEDs added)."""
    parts = SOURCES[name]
    document = copy.deepcopy(read(parts[0][0]))
    names = {ied["name"] for ied in document["ieds"]}
    for path, count in parts[1:]:
        extra = read(path)
        taken = [ied for ied in extra["ieds"] if ied["name"] not in names][:count]
        for ied in taken:
            document["ieds"].append(ied)
            document["scl_sources"][ied["source_id"]] = extra["scl_sources"][ied["source_id"]]
            names.add(ied["name"])
    for ied in document["ieds"]:
        ied["port"], ied["role"] = IED_PORT, "simulated"
    document["name"] = f"Trial {name} ({len(document['ieds'])} IEDs)"
    return document


def archive(document):
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", zipfile.ZIP_DEFLATED) as target:
        target.writestr("project.json", json.dumps(document))
    return stream.getvalue()


def open_in(engine, document):
    """POST /project/open with the document as a .scadabench file; returns the opened project."""
    engine.upload("/api/v1/project/open", "file", "trial.scadabench", archive(document))
    return engine.json("GET", "/api/v1/project")


def lab_address(index):
    return f"{RANGE}.{index + 10}"


def use_lab_card(engine):
    """Put every simulated IED on the Lab card at 198.18.20.<10+n>; {ied_id: ip}."""
    project = engine.json("GET", "/api/v1/project")
    chosen = {ied["id"]: lab_address(index)
              for index, ied in enumerate(project["ieds"]) if ied.get("role") == "simulated"}
    status = engine.json("POST", "/api/v1/network/lab/use", {"addresses": chosen})
    if not status.get("active"):
        raise RuntimeError(f"Lab card not in use after /network/lab/use: {status.get('problem')}")
    return chosen


if __name__ == "__main__":
    for key in SOURCES:
        built = build(key)
        print(key, len(built["ieds"]), "IEDs", len(built["diagram"].get("nodes", [])), "diagram nodes")
