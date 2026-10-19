"""Removal, bulk-removal fingerprint, compact/expand, snapshot and new-project vectors."""
import copy
import hashlib

from studio_app.projects import bulk_removal
from vectors_common import Stub, roundtrip

A, B, C = "a" * 32, "b" * 32, "c" * 32


def sha(text):
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def ied(key, name, source="<SCL/>", **extra):
    return {"id": key, "name": name, "filename": name + ".cid", "source": source, "cfg": "cfg",
            "signals": [{"ref": "x"}] * 3, "ip": "127.50.0.2", "port": 102, "role": "simulated", **extra}


def binding(key, ref="r"):
    return {"ied_id": key, "ref": ref, "report_ref": None}


def data(kind, **extra):
    return {"label": kind, "kind": kind, "state_mapping": None, "control_binding": None, "relay_statuses": [], **extra}


def project():
    return {
        "id": "f" * 32, "name": "Station é \U0001F600", "ieds": [ied(A, "A"), ied(B, "B", source="<SCL b/>"),
                                                                ied(C, "C", source="<SCL/>")],
        "diagram": {"nodes": [
            {"id": "n1", "data": data("CB", state_mapping={"mode": "direct", "source": binding(A)},
                                      control_binding={"ied_id": B, "ref": "c"})},
            {"id": "n2", "data": data("CB", state_mapping={"mode": "contacts_to_dbpos", "open": binding(B),
                                                           "closed": binding(C)})},
            {"id": "n3", "data": data("RELAY", relay_statuses=[{"id": "s1", "label": "x", "source": binding(A)},
                                                               {"id": "s2", "label": "y", "source": binding(C)}])},
            {"id": "n4", "data": data("CT")}], "edges": []},
        "sources": [{"ied_id": A, "ref": "m", "spec": {"mode": "ramp", "offset": 0.0}},
                    {"ied_id": C, "ref": "m", "spec": {"mode": "csv", "csv_text": "t,v", "points": [[0.0, 1.0]]}}],
        "scenarios": [{"id": "s1", "name": "one", "enabled": False, "steps": [{"id": "a", "ied_id": A}, {"id": "b", "ied_id": B}]},
                      {"id": "s2", "name": "two", "enabled": True, "steps": [{"id": "a", "ied_id": A}]}],
        "test_panel": [{"id": "t", "ied_id": B}, {"id": "u", "ied_id": C}],
        "network": {"mode": "loopback", "nominal_hz": 50.0, "x": float("inf")},
        "gateway": {"role": "gateway", "points": {"iec104": [{"ied": A}]}},
    }


def cases(fixtures):
    out = []

    def add(name, fn, **args):
        out.append({"name": name, "fn": fn, "args": args, "ieds": []})

    for ids in ([A], [B], [C], [A, B], [A, B, C], [], ["zz"]):
        add(f"impacts {ids}", "impacts", project=project(), ids=ids)
        add(f"without {ids}", "without_ieds", project=project(), ids=ids)
    no_panel = project()
    del no_panel["test_panel"]
    add("impacts no panel", "impacts", project=no_panel, ids=[C])
    add("without no panel", "without_ieds", project=no_panel, ids=[C])
    for key in (A, B, C, "zz"):
        add(f"removal preview {key}", "removal_preview", project=project(), key=key)
    for keys in ([A], [C, A], [A, B, C], [], [A, A], [A, 5], "x", None, [A] * 21, ["zz"], [B, "zz"],
                 [str(i) for i in range(20)]):
        add(f"bulk preview {keys}", "bulk_preview", project=project(), keys=keys)
    for keys, good in (([A], True), ([A], False), ([C, A], True), ([], True)):
        fingerprint = bulk_removal.preview(Stub(roundtrip(project())), keys)["fingerprint"] if keys and good else "x"
        add(f"bulk remove {keys} {good}", "bulk_remove", project=project(), keys=keys, fingerprint=fingerprint)
    add("compact", "compact", project=project())
    add("compact migration fields", "compact", project={**project(), "format": 3, "migration_notice": {"x": 1},
                                                         "scl_sources": {"x": "y"}})
    add("compact empty", "compact", project={"name": "x"})
    compacted = {"id": "f" * 32, "format": 15, "name": "n", "ieds": [
        {"id": A, "name": "A", "port": 102, "x": 1, "source_id": sha("<SCL/>")},
        {"id": B, "name": "B", "source_id": sha("<SCL/>")}], "scl_sources": {sha("<SCL/>"): "<SCL/>"}, "tail": 1}
    documents = [compacted, {**compacted, "format": 1}, {**compacted, "format": True}, {**compacted, "format": 2.0},
                 {**compacted, "format": 16}, {**compacted, "format": "15"}, {**compacted, "format": [15]},
                 {**compacted, "format": None}, {k: v for k, v in compacted.items() if k != "format"},
                 {**compacted, "scl_sources": []}, {**compacted, "ieds": {}},
                 {**compacted, "scl_sources": {str(i): "x" for i in range(21)}},
                 {**compacted, "ieds": [{"source_id": sha("<SCL/>")}] * 21},
                 {**compacted, "scl_sources": {sha("<SCL/>"): "<SCL />"}},
                 {**compacted, "scl_sources": {sha("<SCL/>"): 5}},
                 {**compacted, "ieds": [5]}, {**compacted, "ieds": [{"source_id": 5}]},
                 {**compacted, "ieds": [{"source_id": "zz"}]}, {**compacted, "ieds": []}]
    for index, document in enumerate(documents):
        add(f"expand {index}", "expand", document=document)
    notice = {"message": "m", "details": ["d"], "backup": ""}
    for index, (note, session, revision) in enumerate(((None, "s" * 32, 0), (notice, "x", 7), ({}, "", 2 ** 40))):
        add(f"snapshot {index}", "snapshot", project=project(), notice=note, session_id=session, revision=revision)
    add("new project", "new_project", id="d" * 32)
    for index, ieds in enumerate(([], [{"ip": ""}, {"ip": "127.50.0.2"}, {}, {"ip": None}, {"ip": 0}],
                                   [{"ip": "127.50.0.%d" % i} for i in range(2, 255)] + [{"ip": ""}],
                                   [{"ip": [1]}, {"ip": ""}], [{"ip": []}, {"ip": {}}], [{"ip": 5}, {"ip": False}])):
        add(f"assign endpoints {index}", "assign_endpoints", ieds=copy.deepcopy(ieds))
    return out
