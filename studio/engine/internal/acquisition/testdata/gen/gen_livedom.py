"""Golden cases for Go package livedom: live_value, model_compare, access_policy, mms_access, int().

Run from this folder with the studio venv:
    ..\\..\\..\\..\\..\\.venv\\Scripts\\python.exe gen_livedom.py
"""
import json
import sys

from common import PUBLIC, project, record

from studio_app.domain.access_policy import choose_access, short_name
from studio_app.domain.live_value import EXPECTED_MMS, convert, describe_mms, type_matches
from studio_app.domain.mms_access import client_view, mark_own_peer, merge_summary
from studio_app.domain.model_compare import compare

NAN, INF = float("nan"), float("inf")


def failure(error):
    return f"{type(error).__name__}: {error}"


def value_cases():
    kinds = list(EXPECTED_MMS) + ["Unknown", "FLOAT", "INT", "", "float32"]
    raws = [None, True, False, 0, 5, -3, 2**70, 1.5, -0.0, 1e300, "text", "", [1, 2], [], {},
            {"access_error": 3}, {"access_error": None}, {"access_error": "x", "bits": 1},
            {"bits": 2, "be": 1, "le": 2}, {"bits": 0, "be": [1], "le": None}, {"octets": "0a0b"},
            {"octets": None, "bits": 1, "be": 1, "le": 0}, {"other": 1}]
    cases = []
    for kind in kinds:
        for index, raw in enumerate(raws):
            try:
                want = json.dumps({"value": convert(kind, raw)})
            except ValueError as error:
                want = failure(error)
            cases.append((f"convert {kind} {index}", ["convert", kind, raw], want))
    codes = [None, True, False, *range(17), 4.0, 5.0, "4", 99, -1, 2**70]
    for kind in kinds:
        for index, code in enumerate(codes):
            cases.append((f"type_matches {kind} {index}", ["type_matches", kind, code],
                          str(type_matches(kind, code))))
    for index, code in enumerate([*range(17), 99, -1, None, True, False, "x", "", 2**70, 4.5, NAN]):
        cases.append((f"describe {index}", ["describe", code], describe_mms(code)))
    return cases


def trimmed(ied):
    return {"signals": [{"ref": s["ref"], "fc": s["fc"], "kind": s["kind"]} for s in ied["signals"]],
            "datasets": [{"ref": d["ref"], "fcda_refs": d.get("fcda_refs", [])} for d in ied["datasets"]],
            "reports": [{"ref": r["ref"], "dataset_ref": r.get("dataset_ref")} for r in ied["reports"]]}


def discovery_of(ied, perturb):
    devices = {}
    for index, signal in enumerate(ied["signals"]):
        ld, rest = signal["ref"].split("/", 1)
        node = devices.setdefault(ld, {}).setdefault(rest.split(".", 1)[0], {"leaves": []})
        if perturb and index % 7 == 3:
            continue
        mms = sorted(EXPECTED_MMS.get(signal["kind"], {8}))[0]
        if perturb and index % 5 == 1:
            mms = 15
        leaf = {"ref": signal["ref"], "fc": signal["fc"], "mms_type": mms}
        if perturb and index % 11 == 2:
            leaf.pop("mms_type")
        if perturb and index % 13 == 4:
            leaf["mms_type"] = float(mms)
        node["leaves"].append(leaf)
        if perturb and index % 9 == 0:
            node["leaves"].append({"ref": signal["ref"] + "x", "fc": signal["fc"], "mms_type": 4})
            node["leaves"].append({"ref": signal["ref"], "fc": "BR", "mms_type": 4})
            node["leaves"].append({"ref": signal["ref"], "fc": "AA", "mms_type": True})
    for dataset in ied["datasets"]:
        owner, name = dataset["ref"].split("$", 1)
        ld, ln = owner.split("/", 1)
        members = [ref + "[ST]" for ref in dataset.get("fcda_refs", [])]
        if perturb and members:
            members = members[1:] + ["X/Y.Z[MX]", "plain"]
        node = devices.setdefault(ld, {}).setdefault(ln, {"leaves": []})
        node.setdefault("dataset_members", {})[name] = {"members": members}
    for index, report in enumerate(ied["reports"]):
        owner, kind, name = report["ref"].rsplit(".", 2)
        ld, ln = owner.split("/", 1)
        if perturb and index == 0:
            continue
        node = devices.setdefault(ld, {}).setdefault(ln, {"leaves": []})
        key = "buffered_reports" if kind == "BR" else "unbuffered_reports"
        node.setdefault(key, []).append(name)
    return {"devices": [{"name": ld, "nodes": [{"name": ln, **node} for ln, node in nodes.items()]}
                        for ld, nodes in devices.items()]}


SYNTHETIC_IED = {
    "signals": [{"ref": "D/XCBR1.Pos.stVal", "fc": "ST", "kind": "Dbpos"},
                {"ref": "D/MMXU1.TotW.mag.f", "fc": "MX", "kind": "FLOAT32"},
                {"ref": "D/GGIO1.Ind1.stVal", "fc": "ST", "kind": "BOOLEAN"},
                {"ref": "D/GGIO1.Ind1.stVal", "fc": "ST", "kind": "BOOLEAN"},
                {"ref": "D/A.b", "fc": "ST", "kind": "Weird"}],
    "datasets": [{"ref": "D/LLN0$Events", "fcda_refs": ["D/XCBR1.Pos.stVal", "D/GGIO1.Ind1.stVal"]},
                 {"ref": "D/LLN0$Gone", "fcda_refs": []}, {"ref": "D/LLN0$Same"}, {"fcda_refs": []}],
    "reports": [{"ref": "D/LLN0.RP.Events01"}, {"ref": "D/LLN0.BR.Missing01"}, {"ref": "D/LLN0.BR.Buf"}],
}
SYNTHETIC_DISCOVERY = {"devices": [{"name": "D", "nodes": [
    {"name": "XCBR1", "leaves": [{"ref": "D/XCBR1.Pos.stVal", "fc": "ST", "mms_type": 3}]},
    {"name": "MMXU1", "leaves": [{"ref": "D/MMXU1.TotW.mag.f", "fc": "MX", "mms_type": 4},
                                 {"ref": "D/MMXU1.TotW.mag.f", "fc": "MX", "mms_type": 6}]},
    {"name": "GGIO1", "leaves": [{"ref": "D/GGIO1.Ind2.stVal", "fc": "ST", "mms_type": 2},
                                 {"ref": "D/A.b", "fc": "ST", "mms_type": 99},
                                 {"ref": "D/A.b", "fc": "MX", "mms_type": None},
                                 {"ref": "D/A.b.c", "fc": "CO", "mms_type": 1},
                                 {"ref": "D/A", "fc": "SV", "mms_type": 1},
                                 {"ref": "D/A.b ", "fc": "ST", "mms_type": 4.0}]},
    {"name": "LLN0", "leaves": [], "dataset_members": {
        "Events": {"members": ["D/XCBR1.Pos.stVal[ST]", "D/GGIO1.Ind2.stVal[ST]", "D/GGIO1.Ind2.stVal[MX]"]},
        "Same": {}, "Gone2": {"members": ["a[b[c"]}},
     "unbuffered_reports": ["Events01"], "buffered_reports": ["Buf", "Extra"]},
]}, {"name": "E"}]}


def compare_cases():
    cases = [("compare synthetic", [SYNTHETIC_IED, SYNTHETIC_DISCOVERY],
              json.dumps(compare(SYNTHETIC_IED, SYNTHETIC_DISCOVERY))),
             ("compare synthetic empty", [SYNTHETIC_IED, {}], json.dumps(compare(SYNTHETIC_IED, {}))),
             ("compare bare ied", [{"signals": []}, SYNTHETIC_DISCOVERY],
              json.dumps(compare({"signals": []}, SYNTHETIC_DISCOVERY)))]
    for name in PUBLIC:
        for ied in project(name)["ieds"]:
            small = trimmed(ied)
            for perturb in (False, True):
                discovery = discovery_of(small, perturb)
                cases.append((f"compare {ied['name']} perturb={perturb}", [small, discovery],
                              json.dumps(compare(small, discovery))))
    return cases


ACCESS_IED = {"reports": [
    {"ref": "D/LLN0.RP.Events01", "dataset_ref": "D/LLN0$Events"},
    {"ref": "D/LLN0.RP.Events02", "dataset_ref": "D/LLN0$Events"},
    {"ref": "D/LLN0.BR.Meas01", "dataset_ref": "D/LLN0$Meas"},
    {"ref": "D/LLN0.BR.Odd", "dataset_ref": "NoDollar"}, {"ref": "D/LLN0.BR.None", "dataset_ref": None},
    {"ref": "D/LLN0.BR.Empty", "dataset_ref": ""}, {"ref": "D/LLN0.BR.Missing"},
    {"ref": "D/LLN0.BR.Alpha", "dataset_ref": "A/LLN0$Alpha"}]}
STATUSES = [
    {},
    {"D/LLN0.RP.Events01": {"readable": True, "free": False, "enabled": True},
     "D/LLN0.RP.Events02": {"readable": True, "free": True},
     "D/LLN0.BR.Meas01": {"readable": True, "free": False, "owned": True}},
    {"D/LLN0.RP.Events01": {"readable": 1, "free": "yes"}, "D/LLN0.RP.Events02": {"readable": True, "free": True},
     "D/LLN0.BR.Meas01": None, "D/LLN0.BR.Odd": {}, "D/LLN0.BR.Alpha": {"readable": False, "free": True}},
    {"D/LLN0.RP.Events01": {"readable": True}, "D/LLN0.RP.Events02": {"free": True},
     "D/LLN0.BR.Odd": {"readable": True, "free": 0}, "D/LLN0.BR.Alpha": {"readable": True, "free": True}},
]


def access_cases():
    cases = [(f"short_name {index}", ["short_name", text], short_name(text))
             for index, text in enumerate(["a.b.c", "abc", "", ".", "a.", "D/LLN0.RP.Events01"])]
    ieds = [ACCESS_IED, {}, {"reports": []}]
    ieds += [{"reports": trimmed(ied)["reports"]} for name in PUBLIC for ied in project(name)["ieds"]]
    for i, ied in enumerate(ieds):
        free = {report["ref"]: {"readable": True, "free": True} for report in ied.get("reports", [])}
        for j, status in enumerate(STATUSES + [free]):
            for access in ("auto_reports", "polling", "other"):
                result = choose_access(ied, status, access)
                cases.append((f"choose {i} {j} {access}", ["choose", ied, status, access], json.dumps(result)))
    return cases


def summary(peer, reads=None, connected=True, rcbs=(), writes=None, total=0):
    return {"peer": peer, "connected": connected, "reads": reads or {}, "writes": writes or {},
            "rcbs": list(rcbs), "total_reads": total}


MANY = {f"LD/N{index:02d}": index % 4 + 1 for index in range(15)}
SCENARIOS = {
    "unit": [("merge", [summary("10.0.0.9:50001", {"IED1LD0/XCBR1": 3, "IED1LD0/LLN0": 1}, total=4)], 100.0),
             ("merge", [summary("10.0.0.9:50001", {"IED1LD0/XCBR1": 2}, rcbs=["LLN0.brcb01"], total=6)], 101.0),
             ("view", 102.0), ("view", 120.0)],
    "peers": [("merge", [summary("10.0.0.2:1", {"A/B": 1}, total=5), summary("127.0.0.1:9", {"A/C": 2}, total=5),
                         summary("[::1]:5000", {}, total=9), summary("nocolon", {"X": 1}, total=0),
                         summary("10.0.0.1:2", {"A/B": 1}, writes={"w1": 2, "w2": 3}, total=5)], 10.0),
              ("own", "127.0.0.1:7000", 11.0), ("own", "10.0.0.2:1", 11.5), ("view", 12.5), ("view", 13.5),
              ("merge", [summary("10.0.0.2:1", connected=False), {"peer": "10.0.0.2:1", "reads": {"Z": 1}}], 14.0),
              ("view", 15.0)],
    "connected": [("merge", [summary("a:1", {"N": 1}), summary("b:1", {"N": 1}), summary("c:1", {"N": 1})], 1.0),
                  ("merge", [{"peer": "a:1", "connected": None}, {"peer": "b:1", "connected": 0},
                             {"peer": "c:1"}, {"peer": ""}, {"peer": None}, {}, {"connected": False}], 2.0),
                  ("merge", [{"peer": "a:1", "connected": "no"}], 3.0), ("view", 3.0)],
    "many": [("merge", [summary("1.1.1.1:1", MANY, total=40)], 0.25), ("view", 5.25), ("view", 5.2500001),
             ("view", 2.75), ("view", 3.5)],
    "ints": [("merge", [summary("p:1", {"A": 2.7, "B": True, "C": "3", "D": " 4 ", "E": -1.5}, writes={"w": "٣"},
                                total="7")], 1.0),
             ("merge", [{"peer": "p:1", "reads": {"A": 1}, "total_reads": 2.9}], 2.0),
             ("merge", [{"peer": "p:1", "reads": {"A": 1, "B": "x", "C": 1}}], 3.0),
             ("merge", [{"peer": "p:1", "reads": {"A": None}}], 4.0),
             ("merge", [{"peer": "p:1", "total_reads": NAN}], 5.0),
             ("merge", [{"peer": "p:1", "total_reads": INF}], 6.0),
             ("merge", [{"peer": "p:1", "total_reads": "1_000"}, {"peer": "q:1", "total_reads": [1]}], 7.0),
             ("view", 8.0)],
    "shapes": [("merge", [{"peer": "r:1", "reads": None, "rcbs": None, "writes": None}], 1.0),
               ("merge", [{"peer": "r:1", "rcbs": ["b", "a", "C"], "reads": {}}], 2.0),
               ("merge", [{"peer": "r:1", "rcbs": "ba"}, {"peer": "s:1", "rcbs": {"y": 1, "x": 2}}], 3.0),
               ("merge", [{"peer": "s:1", "rcbs": [3, 1.5, True]}], 4.0),
               ("merge", [7], 5.0), ("merge", [{"peer": "t:1", "reads": [1]}], 6.0),
               ("merge", [{"peer": 5}], 7.0), ("merge", [{"peer": 5, "connected": False}], 8.0),
               ("own", "r:1", 9.0), ("merge", [{"peer": "r:1", "reads": {"Q": 1}}], 9.5), ("view", 10.0)],
}


def mms_cases():
    cases = []
    for name, steps in SCENARIOS.items():
        clients, outputs = {}, []
        for step in steps:
            try:
                if step[0] == "merge":
                    merge_summary(clients, step[1], step[2])
                    outputs.append(clients)
                elif step[0] == "own":
                    mark_own_peer(clients, step[1], step[2])
                    outputs.append(clients)
                else:
                    outputs.append(client_view(clients, step[1]))
            except Exception as error:  # the Go port stops at the same item with the same text
                outputs.append({"error": failure(error), "clients": clients})
            outputs[-1] = json.loads(json.dumps(outputs[-1]))
        cases.append((f"mms {name}", [[list(step) for step in steps]], json.dumps(outputs)))
    return cases


def int_cases():
    texts = [" 7 ", "+5", "-5", "1_0", "1__0", "_1", "1_", "x", "", " ", "٣", "５", " 5　",
             "\x1c5", "5\x1c", "+ 5", "0x10", "007", "0_0", "-0", "1" * 4300, "1" * 4301, "1_" * 4300 + "1",
             "0" * 5000 + "1", "1" * 5000 + "x", "a" * 300, "é" * 300, "\x00" * 300, "a" * 198, "a" * 199,
             "'", "'\"", "\"", "\x855", "5​", "\U0001d7ce", "1.5", "\t\n5\r\x0b\x0c", "99999999999999999999",
             "-99999999999999999999", "9223372036854775807", "9223372036854775808", "퟿", "๓๔", "--1",
             "5 5", "۱١", "1 ", " -3 ", "+ 5", "\U0001fbf9", "\\", "\x7f", "͸"]
    cases = []
    for index, text in enumerate(texts):
        try:
            want = str(int(text))
        except ValueError as error:
            want = failure(error)
        cases.append((f"int text {index}", ["text", text], want))
    for index, value in enumerate([True, False, 2.9, -2.9, -0.5, 1e18, NAN, INF, -INF, None, [1], {}, 7, -7, "12"]):
        try:
            want = str(int(value))
        except Exception as error:
            want = failure(error)
        cases.append((f"int value {index}", ["value", value], want))
    spaces, digits = [], []
    for code in range(128, sys.maxunicode + 1):
        if 0xD800 <= code <= 0xDFFF:
            continue
        char = chr(code)
        try:
            digits.append([code, int(char)])
        except ValueError:
            try:
                int(char + "5" + char)
                spaces.append(code)
            except ValueError:
                pass
    cases.append(("int spaces", ["spaces", spaces], ""))
    cases.append(("int digits", ["digits", digits], ""))
    return cases


if __name__ == "__main__":
    record("livedom_value.json", value_cases())
    record("livedom_compare.json", compare_cases())
    record("livedom_access.json", access_cases())
    record("livedom_mms.json", mms_cases())
    record("livedom_int.json", int_cases())
