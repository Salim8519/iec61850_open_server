"""ProjectService.restore vectors: "@source:NAME" placeholders stand for the fixture SCL."""
import hashlib

from vectors_common import FIXTURE_IDS, fixture_sources

GUID = "{0123ABCD-0123-4567-89AB-0123456789AB}"
B, B2, F = FIXTURE_IDS["BayController"], FIXTURE_IDS["Bay2"], FIXTURE_IDS["FEED1"]
P = "BayControllerCTRL/"


def item(name, **extra):
    filename = fixture_sources()[name][1]
    return {"id": FIXTURE_IDS[name], "name": name, "filename": filename, "ip": "", "port": 102, **extra}


def compact_doc(version, items, **extra):
    sources = fixture_sources()
    table, ieds = {}, []
    for entry in items:
        digest = hashlib.sha256(sources[entry["name"]][0].encode("utf-8")).hexdigest()
        table[digest] = "@source:" + entry["name"]
        ieds.append({**entry, "source_id": digest})
    return {"id": "f" * 32, "name": "Restored", "format": version, "ieds": ieds, "scl_sources": table, **extra}


def legacy_doc(items, **extra):
    return {"format": 1, "id": "f" * 32, "ieds": [{**entry, "source": "@source:" + entry["name"]} for entry in items],
            **extra}


def rich(version=1):
    """A project whose stored diagram only uses fields its format version had."""
    trip = {"label": "Trip", "kind": "TRIP_RELAY"}
    if version < 7:
        trip["trip_polarity"] = "operate_true"
    nodes = [{"id": "cb", "position": {"x": 1, "y": 2}, "data": {
        "label": "CB", "kind": "CB", "state_mapping": {"mode": "direct", "source": {"ied_id": B, "ref": P + "XCBR1.Pos.stVal"}},
        "control_binding": {"ied_id": B, "ref": P + "XCBR1.Pos", "compatibility": "concen"}}},
        {"id": "trip", "position": {"x": 1, "y": 2}, "data": trip},
        {"id": "btn", "position": {"x": 1, "y": 2}, "data": {"label": "Btn", "kind": "PUSH_BUTTON"}}]
    if version < 12:
        nodes.append({"id": "bus", "position": {"x": 1, "y": 2}, "data": {"label": "Bus", "kind": "BUSBAR", "width": 3}})
    return {
        "diagram": {"nodes": nodes,
                    "edges": [{"id": "w", "source": "cb", "target": "btn"}, {"id": "w2", "source": "cb", "target": "trip"}]},
        "sources": [{"ied_id": B, "ref": P + "GGIO1.AnIn1.mag.f", "spec": {"mode": "sine", "amplitude": 5}}],
        "scenarios": [{"id": "s", "name": " Run ", "steps": [{"id": "a", "at": 2, "ied_id": B, "ref": P + "XCBR1.Loc.stVal", "value": True}]}],
        "test_panel": [{"id": "t", "label": "Loc", "ied_id": B, "ref": P + "XCBR1.Loc.stVal"}],
        "network": {"mode": "lab", "adapter_guid": GUID, "adapter_label": "Eth"},
        "gateway": {"protocol": "modbus", "points": {"modbus": [{"kind": "single", "address": 1, "ied": B, "ref": "x"}]}},
    }


def cases(fixtures):
    out = []

    def add(name, document, new_id="d" * 32):
        out.append({"name": name, "fn": "restore", "args": {"document": document, "new_id": new_id}, "ieds": []})

    two = [item("BayController"), item("Bay2", ip="127.50.0.2")]
    for version in range(2, 16):
        add(f"compact v{version}", compact_doc(version, two, **rich(version)))
        add(f"compact stale v{version}", compact_doc(version, two, **rich()))
        add(f"compact plain v{version}", compact_doc(version, two))
    add("legacy v1", legacy_doc(two, **rich()))
    add("legacy v1 no extras", legacy_doc([item("BayController")]))
    add("current lab with reach", compact_doc(15, two, network={"mode": "lab", "adapter_guid": GUID, "goose_reach": "inside"}))
    add("feed1", compact_doc(15, [item("FEED1", role="real", live={"ip": "10.0.0.1"}), item("BayController")]))
    add("no id", {k: v for k, v in compact_doc(15, two).items() if k != "id"})
    add("bad id", compact_doc(15, two, id="F" * 32))
    add("int id", compact_doc(15, two, id=5))
    add("long name", compact_doc(15, two, name="é" * 120))
    add("name number", compact_doc(15, two, name=12.5))
    add("ieds not list", {"format": 1, "ieds": {}})
    add("too many ieds", legacy_doc([item("BayController")] * 21))
    add("ied not dict", {"format": 1, "ieds": [5]})
    add("ied bad id", legacy_doc([item("BayController", id="xyz")]))
    add("ied int id", legacy_doc([item("BayController", id=5)]))
    add("ied no source", {"format": 1, "ieds": [item("BayController")]})
    add("ied list source", {"format": 1, "ieds": [{**item("BayController"), "source": [1]}]})
    add("duplicate ids", legacy_doc([item("BayController"), item("Bay2", id=B)]))
    add("duplicate names", legacy_doc([item("BayController"), item("BayController", id="9" * 32)]))
    nameless = {k: v for k, v in item("BayController").items() if k != "name"}
    add("no name", {"format": 1, "ieds": [{**nameless, "source": "@source:BayController"}]})
    for index, extra in enumerate([{"port": "abc"}, {"port": None}, {"port": "103"}, {"port": 1e3}, {"port": float("inf")},
                                   {"port": 10 ** 20}, {"role": "x"}, {"role": ["x"]}, {"live": [1]},
                                   {"live": {"ip": "999.1.1.1"}}, {"lab_ip": "127.0.0.1"}, {"test_mode": "x"},
                                   {"goose_publish": None}, {"ip": "10.0.0.7"}, {"ip": 0}, {"ip": [1]},
                                   {"role": "real", "live": {"access": "polling", "poll_ms": 500}}]):
        add(f"ied settings {index}", compact_doc(15, [item("BayController", **extra), item("Bay2")]))
    for index, extra in enumerate([{"network": [1]}, {"network": {"mode": "x"}}, {"network": {"nominal_hz": [1]}},
                                   {"gateway": []}, {"gateway": {"protocol": "x"}}, {"gateway": {"settings": [1]}},
                                   {"diagram": None}, {"diagram": {"nodes": [{}]}}, {"sources": None},
                                   {"scenarios": None}, {"test_panel": None}, {"diagram": {"nodes": [5]}}]):
        add(f"section {index}", compact_doc(15, two, **extra))
        add(f"legacy section {index}", compact_doc(9, two, **extra))
    add("same name twice", compact_doc(15, [item("BayController"), item("Bay2"), item("Bay2", id="4" * 32)]))
    return out
