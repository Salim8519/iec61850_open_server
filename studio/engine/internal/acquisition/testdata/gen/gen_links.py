"""Golden cases for the GOOSE link map in Go package goosedom (build_link_map, with_live).

Writes links_map.json ([declarations, project_ieds]) and links_live.json ([link_map,
watches, running]). Declarations are written as the Python dict with "blocks" as
[[key list, block], ...] (JSON has no tuple keys). Real declarations come from
read_goose_sources over the public projects; synthetic ones cover every branch.
`want` is json.dumps of the Python result, compared with pyjson.Dumps in Go.
"""
import copy
import json

from common import project, record
from studio_app.domain.goose_link_live import with_live
from studio_app.domain.goose_link_map import build_link_map
from studio_app.projects.scl_sources import stored_sources
from studio_app.scl.goose_sources import read_goose_sources


def layout(declarations):
    return {**declarations, "blocks": [[list(key), block] for key, block in declarations["blocks"].items()]}


def project_ieds(ieds):
    return [{"id": ied["id"], "name": ied["name"], "role": ied.get("role", "simulated")} for ied in ieds]


def real_declarations(name):
    loaded = project(name, keep_source=True)
    sources = list(stored_sources(loaded).values())
    return read_goose_sources([(entry["text"], entry["ieds"]) for entry in sources]), project_ieds(loaded["ieds"])


def fcda(ld="LD0", ln="GGIO", inst="1", do="Ind1", da="stVal", fc="ST", prefix=""):
    item = {"ldInst": ld, "prefix": prefix, "lnClass": ln, "lnInst": inst, "doName": do, "fc": fc}
    if da is not None:
        item["daName"] = da
    return item


def block(publisher, name, status="sendable", app_id=4097, mac="01-0C-CD-01-00-01", ld="LD0", dataset="ds",
          fcdas=None):
    domain = publisher + ld
    return {"publisher": publisher, "ld_inst": ld, "name": name, "gocb_ref": f"{domain}/LLN0$GO${name}",
            "go_id": name, "dataset": dataset, "dataset_ref": f"{domain}/LLN0${dataset}" if dataset else "",
            "conf_rev": 1, "app_id": app_id, "mac": mac, "vlan_id": 0, "status": status, "label": name.upper(),
            "vendor": {"x": 1}, "fcdas": [fcda()] if fcdas is None else fcdas}


def subscription(publisher, name, status="sendable", matches=("named",)):
    return {"gocb_ref": f"{publisher}LD0/LLN0$GO${name}", "publisher": publisher, "name": name, "status": status,
            "lgos": None, "member_count": 1,
            "inputs": [{"int_addr": f"in{index}", "desc": f"d{index}", "source": f"{publisher}/LD0/GGIO1.Ind1",
                        "match": match, "ld": "SLD", "member": 0, "member_path": ""}
                       for index, match in enumerate(matches)]}


def remote(publisher, ld="LD0", cb="cbR", hint=None, int_addr="x"):
    return {"publisher": publisher, "ld_inst": ld, "cb_name": cb, "int_addr": int_addr, "desc": "",
            "source": f"{publisher}/{ld}/GGIO1.Ind1.stVal", "hint": hint}


def synthetic():
    fcdas = [fcda(), fcda(ld="LD9", da=None, prefix="Q"), fcda(fc="MX", do="AnIn1", da="mag.f"),
             {"lnClass": "LLN0", "doName": "Mod"}, fcda(ld="LD0", do="", da="")]
    blocks = {("P1", "LD0", "zz"): block("P1", "zz", app_id=0), ("P1", "LD0", "cb1"): block("P1", "cb1", fcdas=fcdas),
              ("P1", "LD0", "cb2"): block("P1", "cb2", status="disabled_in_file"),
              ("P1", "LD0", "cb3"): block("P1", "cb3", status="no_dataset", dataset=""),
              ("P1", "LD1", "cb4"): block("P1", "cb4", status="no_address", app_id=None, mac="", ld="LD1"),
              ("P1", "LD0", "cb5"): block("P1", "cb5", status="not_goose"),
              ("P1", "LD0", "cb6"): block("P1", "cb6", status="not_sendable", dataset="", fcdas=fcdas),
              ("P2", "LD0", "a1"): block("P2", "a1", app_id=4097, mac="01-0C-CD-01-00-02"),
              ("P2", "LD0", "a2"): block("P2", "a2", app_id=0x1FF, mac="01-0C-CD-01-00-02"),
              ("P2", "LD0", "a3"): block("P2", "a3", app_id=0x1FF, mac=""),
              ("P2", "", "a4"): block("P2", "a4", app_id="", mac=None, ld=""),
              ("Z9", "LD0", "x"): block("Z9", "x", app_id=0)}
    for number in range(7):
        blocks[("P3", "LD0", f"z{number}")] = block("P3", f"z{number}", app_id=0, mac=f"01-0C-CD-01-01-0{number}")
    subscriptions = {
        "S1": [subscription("P1", "cb1"), subscription("P1", "cb2", "disabled_in_file", ("named", "data")),
               subscription("P1", "cb3", "no_dataset", ()), subscription("P2", "a1", matches=("data", "data")),
               subscription("P1", "cb4", "no_address"), subscription("P3", "z0")],
        "S2": [subscription("P1", "cb5", "not_goose"), subscription("P1", "cb6", "not_sendable"),
               subscription("P1", "cb1")],
        "Z8": [subscription("P1", "zz")], "Z7": [subscription("Z6", "q")], "P1": [subscription("P2", "a2")]}
    hint = {"mac": "01-0C-CD-01-00-99", "publisher": "R1"}
    remotes = {"S1": [remote("R1", hint=None), remote("R1", hint=hint), remote("R1", cb="", hint=hint),
                      remote("R1", cb="", hint=None), remote("R2", ld="", cb="cbX"), remote("R1", ld=None, cb=None)],
               "S2": [remote("R3", hint={"mac": ""}), remote("R3", hint={"mac": "01-0C-CD-01-00-98"})],
               "Z5": [remote("R9")], "S9": []}
    ieds = {name: {"desc": f"{name} desc", "manufacturer": "ACME", "type": "T", "loaded": True}
            for name in ("P1", "P2", "P3", "S1", "S2", "Z9", "Z8", "Z7")}
    ieds["S2"] = {}
    return {"ieds": ieds, "blocks": blocks, "subscriptions": subscriptions,
            "problems": {"S1": ["Input in9 expects GOOSE block P9/LD0/cb, which is not a publishable GOOSE block",
                                "Input from P9/LD0 matches no GOOSE block; add srcCBName to the ExtRef"],
                         "Z9": ["not involved"], "P1": []},
            "remote": remotes}


PROJECT = [{"id": "i1", "name": "S1", "role": "simulated"}, {"id": "i2", "name": "P1", "role": "real"},
           {"id": "i3", "name": "Q1"}, {"id": "i4", "name": "S2", "role": "off"}]


def map_cases():
    cases, maps = [], []

    def add(name, declarations, ieds):
        result = build_link_map(copy.deepcopy(declarations), copy.deepcopy(ieds))
        cases.append((name, [layout(declarations), ieds], json.dumps(result)))
        maps.append((name, result))
    for name in ("open", "model"):
        declarations, ieds = real_declarations(name)
        add(f"build_link_map/{name}", declarations, ieds)
        add(f"build_link_map/{name}/first_only", declarations, ieds[:1])
        add(f"build_link_map/{name}/none", declarations, [])
    empty = {"ieds": {}, "blocks": {}, "subscriptions": {}, "problems": {}, "remote": {}}
    add("build_link_map/empty", empty, PROJECT)
    declarations = synthetic()
    add("build_link_map/synthetic", declarations, PROJECT)
    add("build_link_map/synthetic/s2_only", declarations, PROJECT[3:])
    add("build_link_map/synthetic/p2", declarations, [{"id": "x", "name": "P2"}, {"id": "y", "name": "P2"}])
    add("build_link_map/synthetic/unknown", declarations, [{"id": "u", "name": "U1", "role": None}])
    record("links_map.json", cases)
    return maps


def watches_for(link_map):
    refs = [link["gocb_ref"] for link in link_map["links"] if link["status"] == "declared"]
    states = [None, "stopped", "waiting", "timeout", "problem", "receiving"]
    watches = []
    for index, ref in enumerate(refs):
        state = states[index % len(states)]
        watch = {"gocb_ref": ref, "st_num": index, "sq_num": None, "observed_at": "2026-10-20T10:00:00",
                 "frames": index * 10, "problem": "conf_rev_mismatch" if state == "problem" else None}
        if state is not None:
            watch["state"] = state
        watches.append(watch)
    return watches


def live_cases(maps):
    cases = []
    for name, link_map in maps:
        watches = watches_for(link_map)
        variants = {"watched": watches, "none": [], "shifted": watches[1:] + watches[:1],
                    "duplicate": watches + [{**item, "state": "receiving", "st_num": 99} for item in watches]}
        for variant, items in variants.items():
            for running in (True, False):
                args = [link_map, items, running]
                result = with_live(*copy.deepcopy(args))
                cases.append((f"with_live/{name}/{variant}/{running}", args, json.dumps(result)))
    record("links_live.json", cases)


if __name__ == "__main__":
    live_cases(map_cases())
