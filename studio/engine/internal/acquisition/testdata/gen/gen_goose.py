"""Golden cases for Go package goosedom: supervision, inputs, reach and adapter order.

Writes goose_supervision.json, goose_inputs.json, goose_reach.json and goose_adapters.json.
`want` is json.dumps of the Python result (insertion order kept), compared with
pyjson.Dumps in Go. Inputs are the public projects plus synthetic dicts for every branch.
"""
import copy
import json

from common import project, record
from studio_app.domain import goose_inputs, goose_reach, goose_supervision
from studio_app.networking import adapters as net


def case(name, function, *args):
    return name, list(args), json.dumps(function(*copy.deepcopy(args)))


def slim(ied, keys=("id", "name", "role", "goose")):
    return {key: ied[key] for key in keys if key in ied}


def signals_of(ied):
    return [{key: signal.get(key) for key in ("ref", "fc", "kind", "writable")} for signal in ied["signals"]]


def supervision_cases():
    cases = []
    for name in ("open", "model"):
        ieds = [slim(ied) for ied in project(name)["ieds"]]
        cases.append(case(f"watch_list/{name}", goose_supervision.watch_list, ieds))
        for role in ("real", "off", None, "outside"):
            changed = copy.deepcopy(ieds)
            changed[-1]["role"] = role
            cases.append(case(f"watch_list/{name}/last_{role}", goose_supervision.watch_list, changed))
        missing = copy.deepcopy(ieds)
        missing[0].pop("role")
        missing[0]["goose"] = None
        cases.append(case(f"watch_list/{name}/no_goose", goose_supervision.watch_list, missing))
    control = {"gocb_ref": "P/LLN0$GO$a", "app_id": 1, "mac": "01-0C-CD-01-00-01", "go_id": "a",
               "dataset_ref": "P/LLN0$ds", "conf_rev": 1}
    synthetic = [
        {"id": "s1", "name": "S1", "goose": {"subscribes": [{**control, "publisher": "P", "lgos": "S1LD/LGOS1",
                                                             "inputs": [{}, {}]},
                                                            {**control, "gocb_ref": "X/LLN0$GO$x"}]}},
        {"id": "p1", "name": "P", "role": "simulated", "goose": {"publishes": [control, dict(control)]}},
        {"id": "s2", "name": "S2", "goose": {"subscribes": [{**control, "inputs": []}]}},
        {"id": "r1", "name": "R", "role": "real", "goose": {"publishes": [{**control, "gocb_ref": "R/LLN0$GO$r"}],
                                                            "subscribes": [{**control, "gocb_ref": "Q"}]}},
    ]
    cases.append(case("watch_list/synthetic", goose_supervision.watch_list, synthetic))
    cases.append(case("watch_list/empty", goose_supervision.watch_list, []))
    cases += lgos_cases()
    cases += dataset_cases()
    record("goose_supervision.json", cases)


def lgos_cases():
    ln = "S1LD/LGOS1"
    full = {f"{ln}.St.stVal": {"kind": "BOOLEAN"}, f"{ln}.SimSt.stVal": {"kind": "BOOLEAN"},
            f"{ln}.LastStNum.stVal": {"kind": "INT32U"}, f"{ln}.ConfRevNum.stVal": {"kind": "INT32U"}}
    as_bool = {key: {"kind": "BOOLEAN"} for key in full}
    as_int = {key: {"kind": "INT32"} for key in full}
    partial = {f"{ln}.St.stVal": {"kind": "BOOLEAN"}, f"{ln}.LastStNum.stVal": None}
    model = project("model")["ieds"][1]
    real = {signal["ref"]: {"kind": signal["kind"]} for signal in model["signals"][:40]}
    received = [{"state": "receiving", "test": True, "st_num": 5, "conf_rev": 2},
                {"state": "timeout", "test": None, "st_num": 0, "conf_rev": None},
                {"state": "receiving", "test": 1, "st_num": 3.0, "conf_rev": True},
                {"st_num": "7", "conf_rev": 12345678901234},
                {}]
    cases = []
    for index_name, index in (("full", full), ("bool", as_bool), ("int", as_int), ("partial", partial),
                              ("model", real), ("empty", {})):
        for number, item in enumerate(received):
            cases.append(case(f"lgos_updates/{index_name}/{number}", goose_supervision.lgos_updates, ln, item, index))
    for lgos in (None, "", "BUS1LD1/LLN0"):
        cases.append(case(f"lgos_updates/lgos_{lgos}", goose_supervision.lgos_updates, lgos, received[0], full))
    return cases


def dataset_cases():
    cases = []
    for name in ("open", "model"):
        for ied in project(name)["ieds"]:
            small = {"datasets": ied["datasets"], "signals": signals_of(ied)}
            for control in ied["goose"]["publishes"]:
                count = len(control["members"])
                tag = f"dataset_values/{ied['name']}"
                cases.append(case(tag, goose_supervision.dataset_values, small, control,
                                  [index % 2 == 0 for index in range(count)]))
                cases.append(case(f"{tag}/short", goose_supervision.dataset_values, small, control, [True] * (count - 1)))
                cases.append(case(f"{tag}/not_list", goose_supervision.dataset_values, small, control, {"a": 1}))
                cases.append(case(f"{tag}/no_dataset", goose_supervision.dataset_values, small,
                                  {**control, "dataset_ref": "nope"}, [1] * count))
    feed = project("model")["ieds"][0]
    signals = signals_of(feed)
    structured = [["FEED1LD1/XCBR1.Pos", "XCBR1$ST$Pos"], ["FEED1LD1/XSWI2.Pos", "LD1/XSWI2$CO$Pos"],
                  ["FEED1LD1/XSWI3.Pos.stVal", "XSWI3"], ["FEED1LD1/LLN0.Mod", "LLN0$ST$Mod"],
                  ["S/A.B", "A$ST$B"]]
    signals += [{"ref": "S/A.B(0)", "fc": "ST"}, {"ref": "S/A.B.c", "fc": "ST"}, {"ref": "S/A.BC", "fc": "ST"}]
    refs = [item[0] for item in structured]
    ied = {"datasets": [{"ref": "D", "fcda_refs": refs}], "signals": signals}
    control = {"dataset_ref": "D", "members": [item[1] for item in structured]}
    for name, fc in (("pos", "ST"), ("co", "CO")):
        count = len([s for s in signals if s["fc"] == fc and s["ref"].startswith(refs[0 if fc == "ST" else 1] + ".")])
        nested = [[index, [index + 0.5]] for index in range(count // 2)] + ([7] if count % 2 else [])
        values = [nested, nested, "x", [None], [[1], [2]]]
        cases.append(case(f"dataset_values/structured_{name}", goose_supervision.dataset_values, ied, control, values))
    cases.append(case("dataset_values/structured_mismatch", goose_supervision.dataset_values, ied, control,
                      [[1], [[2, 3]], 4, [], [1, 2, 3]]))
    cases.append(case("dataset_values/members_short", goose_supervision.dataset_values, ied,
                      {**control, "members": control["members"][:2]}, [1, 2, 3, 4, 5]))
    cases.append(case("dataset_values/none_values", goose_supervision.dataset_values, ied, control, None))
    return cases


SIGNALS = [{"ref": "S1LD/GGIO1.Ind1.stVal", "fc": "ST", "kind": "BOOLEAN", "writable": True},
           {"ref": "S1LD/GGIO1.Ind2.stVal", "fc": "ST", "kind": "BOOLEAN", "writable": False},
           {"ref": "S1LD/GGIO1.Mod.ctlModel", "fc": "CF", "kind": "Enum", "writable": True},
           {"ref": "S1LD/GGIO1.AnIn1.mag.f", "fc": "MX", "kind": "FLOAT32", "writable": True},
           {"ref": "S2LD/GGIO1.Ind1.stVal", "fc": "ST", "kind": "BOOLEAN", "writable": True},
           {"ref": "S1LD/GGIO1.Ind3.stVal", "fc": "ST", "kind": "Dbpos", "writable": 1},
           {"ref": "S1LD/GGIO1.Ind3.stVal", "fc": "ST", "kind": "INT32", "writable": True},
           {"ref": "S2LD/XCBR1.Pos.stVal", "fc": "ST", "kind": "Dbpos", "writable": True}]

INPUTS = [{"int_addr": "S1LD/GGIO1.Ind1.stVal", "desc": "Full ref", "member": 0, "source": "P/LD/GGIO1.Ind1.stVal"},
          {"int_addr": "GGIO1.Ind3.stVal", "ld": "S1LD", "member": 1, "member_path": "q",
           "source": "P/LD/GGIO1.Ind3"},
          {"int_addr": "XCBR1.Pos.stVal", "member": 2, "source": "P/LD"},
          {"int_addr": "GGIO1.Ind1.stVal", "member": 0, "desc": ""},
          {"int_addr": "Nope.stVal", "member": 0, "source": "P/LD/x"},
          {"int_addr": "S1LD/GGIO1.Ind2.stVal", "member": 3},
          {"int_addr": "S1LD/GGIO1.Mod.ctlModel", "member": 3, "desc": "cf"},
          {"int_addr": "a b", "member": 0}, {"int_addr": "a|b", "member": 0}, {"int_addr": "", "member": 0},
          {"member": 0, "source": "P/LD/only"}, {"int_addr": "S1LD/GGIO1.AnIn1.mag.f", "member": None},
          {"desc": "No member", "source": "P/LD/y"}, {"int_addr": "GGIO1.Ind3.stVal", "ld": None, "member": 4},
          {"int_addr": "S1LD/GGIO1.AnIn1.mag.f", "member": -1, "member_path": "", "source": "P/LD/GGIO1.AnIn1"}]


def input_cases():
    cases = []
    ied = {"signals": SIGNALS}
    for name, ref in (("named", "P/LD/LLN0$ds"), ("none", None), ("bare", "PLD")):
        subscription = {"gocb_ref": "P/LLN0$GO$cb", "dataset_ref": ref, "inputs": INPUTS}
        cases.append(case(f"bind_inputs/{name}", goose_inputs.bind_inputs, ied, subscription))
    model = project("model")["ieds"][1]
    for number, subscription in enumerate(model["goose"]["subscribes"]):
        cases.append(case(f"bind_inputs/model/{number}", goose_inputs.bind_inputs,
                          {"signals": signals_of(model)}, subscription))
    tweaked = copy.deepcopy(model["goose"]["subscribes"][0])
    tweaked["inputs"][0]["int_addr"] = "PDIF1.Op.general"
    tweaked["inputs"][1]["int_addr"] = "BUS1LD1/LLN0.Mod.stVal"
    cases.append(case("bind_inputs/model/tweaked", goose_inputs.bind_inputs, {"signals": signals_of(model)}, tweaked))
    bindings, _ = goose_inputs.bind_inputs(ied, {"gocb_ref": "g", "dataset_ref": "P/LD/LLN0$ds", "inputs": INPUTS})
    extra = [{**bindings[0], "member": 9}, {**bindings[0], "kind": "FLOAT32", "member": 2},
             {**bindings[0], "kind": "Dbpos", "member": 5}, {**bindings[0], "kind": "INT32", "member": 6},
             {**bindings[0], "member": True}, {**bindings[0], "member": -2}]
    all_bindings = bindings + extra
    frames = [[True, [1, 2], {"bits": 2, "be": 2, "le": 1}, 3, None, {"bits": 1, "be": 1, "le": 2}, 7],
              [False, 5, None, [0.5], 1.5, "x", 2.5], [None, None], [0, 1]]
    for number, values in enumerate(frames):
        for name, published in (("none", None), ("known", {"P/GGIO1.Ind3": 2, "P/GGIO1.AnIn1": 1.25}),
                                ("empty", {})):
            cases.append(case(f"input_updates/{number}/{name}", goose_inputs.input_updates,
                              all_bindings, values, published))
    events = [{"state": "timeout"}, {"state": "receiving", "problem": "conf_rev_mismatch", "values": [1]},
              {"state": "receiving", "values": "x"}, {"state": "receiving", "values": [1, 2]},
              {"state": "receiving", "values": []}, {}, {"state": None, "problem": ""}, {"state": "waiting", "problem": 0}]
    for number, event in enumerate(events):
        for count in (0, 2, 3):
            cases.append(case(f"accept_frame/{number}/{count}", goose_inputs.accept_frame, event, count))
    record("goose_inputs.json", cases)


def adapter(guid, name, kind="ethernet", status="Up", default_route=False, description=""):
    return {"guid": guid, "name": name, "description": description, "kind": kind, "status": status,
            "default_route": default_route}


ADAPTERS = [adapter("{A}", "Ethernet 2", default_route=True), adapter("{B}", "ethernet"),
            adapter("{C}", "Ethernet down", status="Disconnected"), adapter("{D}", "Wi-Fi", kind="wifi"),
            adapter("{E}", "Corp VPN", kind="vpn"), adapter("{F}", "Bluetooth PAN", kind="bluetooth"),
            adapter("{G}", "Ethernet 9", description="Npcap Loopback Adapter"),
            adapter("{H}", "", description="Intel(R) Ethernet I219"), adapter("{I}", None, status="Disconnected"),
            adapter("{J}", "ÉTHERNET Σ", default_route=1), {"guid": "{K}"}]


def reach_cases():
    cases = [case(f"is_npcap_loopback/{index}", goose_reach.is_npcap_loopback, item)
             for index, item in enumerate(ADAPTERS + [{"name": "NPCAP", "description": "LoopBack"}, {}])]
    lists = {"all": ADAPTERS, "no_default": [a for a in ADAPTERS if not a.get("default_route") and a.get("name")],
             "unwired": ADAPTERS[3:6], "none": [],
             "ties": [adapter("{X}", "b", default_route=True), adapter("{Y}", "B", default_route=True),
                      adapter("{Z}", "a")]}
    networks = {"empty": {}, "chosen": {"goose_uplink_guid": "{D}"}, "chosen_gone": {"goose_uplink_guid": "{Q}"},
                "lab": {"mode": "lab", "adapter_guid": "{C}"}, "lab_gone": {"mode": "lab", "adapter_guid": "{Q}"},
                "lab_none": {"mode": "lab", "adapter_guid": None}, "pc_mode": {"mode": "pc", "adapter_guid": "{C}"},
                "chosen_none": {"goose_uplink_guid": None, "mode": "lab", "adapter_guid": "{E}"}}
    for list_name, items in lists.items():
        cases.append(case(f"advanced_choices/{list_name}", goose_reach.advanced_choices, items))
        for net_name, network in networks.items():
            cases.append(case(f"pick_uplink/{list_name}/{net_name}", goose_reach.pick_uplink, items, network))
    uplinks = {"wired": goose_reach.pick_uplink(ADAPTERS, {}), "wifi": goose_reach.pick_uplink(ADAPTERS, networks["chosen"]),
               "none": goose_reach.pick_uplink([], {}), "gone": goose_reach.pick_uplink(ADAPTERS, networks["chosen_gone"]),
               "lab": goose_reach.pick_uplink(ADAPTERS, networks["lab"]), "bare": {"guid": "{A}", "label": None}, "blank": {}}
    risk_networks = [{}, {"mode": "lab", "adapter_guid": "{C}"}, {"mode": "lab", "adapter_guid": "{A}"}]
    for up_name, uplink in uplinks.items():
        for number, network in enumerate(risk_networks):
            for heard in (False, True):
                for ieds in ([], [{"role": "simulated"}, {"role": "real"}], [{}]):
                    cases.append(case(f"share_risk/{up_name}/{number}/{heard}/{len(ieds)}", goose_reach.share_risk,
                                      network, uplink, ieds, heard))
        for reach in ({}, {"goose_reach": "inside"}, {"goose_reach": "outside"},
                      {"goose_reach": "outside", "goose_share_guid": "{A}"},
                      {"goose_reach": "outside", "goose_share_guid": "{D}"}):
            for npcap in (False, True):
                cases.append(case(f"share_hold/{up_name}/{json.dumps(reach)}/{npcap}", goose_reach.share_hold,
                                  reach, uplink, npcap))
        for hold in ("", "Held for a reason."):
            for link in (None, {}, {"share": "sharing"}, {"share": "checking"}, {"share": "off", "uplink": "open"},
                         {"uplink": "closed"}, {"share": None, "uplink": "open"}):
                cases.append(case(f"reach_status/{up_name}/{bool(hold)}/{json.dumps(link)}", goose_reach.reach_status,
                                  {}, uplink, hold, link))
    record("goose_reach.json", cases)


RECORDS = [
    {"guid": "4d36e972-e325-11ce-bfc1-08002be10318", "name": "Ethernet", "description": "Intel(R) Ethernet I219-LM",
     "status": "Up", "index": 7, "mac": "AA-BB", "speed": "1 Gbps", "default_route": True,
     "addresses": {"ip": "192.168.1.10", "prefix": 24, "skip_as_source": False, "origin": "Dhcp"}},
    {"guid": "{abc}", "name": "Wi-Fi", "description": "Wireless 802.11ac", "status": "Up", "default_route": 1,
     "addresses": [{"ip": "10.0.0.2", "prefix": 8}, {"ip": "10.0.0.3", "skip_as_source": 1, "origin": "Manual"}]},
    {"guid": "{LAB}", "name": "Loopback", "description": "Microsoft KM-TEST Loopback Adapter", "status": "Up"},
    {"guid": "x1", "name": "vEthernet (WSL)", "description": "Hyper-V Virtual Ethernet Adapter", "status": "Up"},
    {"guid": "x2", "name": "VMware Network Adapter VMnet8", "description": "VMware Virtual Ethernet Adapter"},
    {"guid": "x3", "name": "WireGuard", "description": "WireGuard Tunnel", "status": "Up", "addresses": None},
    {"guid": "x4", "name": "OpenVPN", "description": "TAP-Windows Adapter V9", "status": "Disconnected"},
    {"guid": "x5", "name": "Bluetooth Network Connection", "description": "Bluetooth Device (PAN)", "status": "Up"},
    {"guid": "x6", "name": "ZeroTier One", "description": "ZeroTier Virtual Port", "status": "Up", "addresses": []},
    {"guid": "x7", "name": "ethernet 3", "description": "Realtek USB", "status": "Disconnected", "default_route": False},
    {"guid": "x8", "name": "Ethernet 2", "description": "Realtek PCIe", "status": "Up", "default_route": False},
    {"guid": "x9", "name": "Ethernet 10", "description": "Realtek PCIe", "status": "Up", "default_route": True},
    {"guid": None, "name": "WLAN", "description": "", "status": "Up", "index": None},
    {"name": "Npcap Loopback Adapter", "description": "Npcap Loopback Adapter", "status": "Up"},
    {"guid": 5, "name": "Ärger", "description": "virtualbox host-only", "status": "Up"},
]


def adapter_cases():
    cases = [case("normalize/all", net.normalize, RECORDS), case("normalize/empty", net.normalize, []),
             case("normalize/reversed", net.normalize, RECORDS[::-1])]
    cases += [case(f"normalize/one/{index}", net.normalize, [item]) for index, item in enumerate(RECORDS)]
    record("goose_adapters.json", cases)


if __name__ == "__main__":
    supervision_cases()
    input_cases()
    reach_cases()
    adapter_cases()
