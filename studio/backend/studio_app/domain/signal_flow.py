"""Build the declared signal-flow graph (GOOSE and reports) of one SCL document.

Pure and deterministic. Every edge comes from an explicit SCL declaration; anything
that cannot be resolved is a diagnostic, never a guessed edge.
"""
import hashlib
import re


def _id(*parts):
    return hashlib.sha256("|".join(parts).encode()).hexdigest()[:16]


def _goose(flow, names, blocks, edges, diagnostics):
    subscribed = set()
    for subscriber, (subscriptions, problems) in sorted(flow["subscriptions"].items()):
        for message in problems:
            diagnostics.append({"level": "problem", "ied": subscriber, "message": message})
        for item in subscriptions:
            subscribed.add(item["gocb_ref"])
            matches = {entry["match"] for entry in item["inputs"]}
            edges.append({"id": _id("goose", item["gocb_ref"], subscriber), "type": "goose",
                          "publisher": item["publisher"], "block": _id("block", item["gocb_ref"]),
                          "subscriber": subscriber, "match": "named" if matches == {"named"} else "data",
                          "signals": [{"source": entry["source"], "int_addr": entry["int_addr"],
                                       "desc": entry["desc"]} for entry in item["inputs"]]})
    for control in flow["gocbs"]:
        blocks.append({"id": _id("block", control["gocb_ref"]), "type": "goose", "ied": control["publisher"],
                       "ref": control["gocb_ref"], "name": control["name"], "dataset": control["dataset"],
                       "app_id": control["app_id"], "members": control["members"]})
        if control["gocb_ref"] not in subscribed:
            diagnostics.append({"level": "info", "ied": control["publisher"],
                                "message": f"Nobody in this file subscribes to GOOSE {control['gocb_ref']}"})


def _reports(flow, names, blocks, edges, diagnostics):
    for report in flow["reports"]:
        block = _id("block", report["ref"])
        blocks.append({"id": block, "type": "report", "ied": report["publisher"], "ref": report["ref"],
                       "name": report["name"], "dataset": report["dataset"], "buffered": report["buffered"],
                       "members": report["members"]})
        if report["dataset"] and not report["dataset_found"]:
            diagnostics.append({"level": "problem", "ied": report["publisher"],
                                "message": f"Report {report['ref']} uses dataset {report['dataset']}, "
                                           "which is not in this file"})
        for client in report["clients"]:
            if client["iedName"] not in names:
                diagnostics.append({"level": "info", "ied": report["publisher"],
                                    "message": f"Report {report['ref']} has client {client['iedName'] or '(unnamed)'}"
                                               ", which is outside this file (for example a gateway or HMI)"})
                continue
            edges.append({"id": _id("report", report["ref"], client["iedName"]), "type": "report",
                          "publisher": report["publisher"], "block": block, "subscriber": client["iedName"],
                          "match": "named",
                          "signals": [{"source": member, "int_addr": "", "desc": ""} for member in report["members"]]})


def build_graph(flow, loaded, ied=None):
    """Return {ieds, blocks, edges, diagnostics, summary}, optionally only around one IED."""
    names = {item["name"] for item in flow["ieds"]}
    blocks, edges, diagnostics = [], [], []
    _goose(flow, names, blocks, edges, diagnostics)
    _reports(flow, names, blocks, edges, diagnostics)
    edges = sorted({edge["id"]: edge for edge in edges}.values(), key=lambda e: (e["type"], e["publisher"], e["id"]))
    if ied is not None:
        edges = [edge for edge in edges if ied in (edge["publisher"], edge["subscriber"])]
        used = {edge["block"] for edge in edges}
        blocks = [block for block in blocks if block["id"] in used or block["ied"] == ied]
        named = re.compile(rf"(?<![A-Za-z0-9_]){re.escape(ied)}(?![A-Za-z0-9_])")
        diagnostics = [item for item in diagnostics if item["ied"] == ied or named.search(item["message"])]
    blocks = sorted(blocks, key=lambda b: (b["type"], b["ied"], b["ref"]))
    ieds = [{**item, "loaded": item["name"] in loaded} for item in sorted(flow["ieds"], key=lambda i: i["name"])]
    if ied is not None:
        involved = {ied} | {edge["publisher"] for edge in edges} | {edge["subscriber"] for edge in edges}
        ieds = [item for item in ieds if item["name"] in involved]
    summary = {"ieds": len(ieds), "goose_links": sum(e["type"] == "goose" for e in edges),
               "report_links": sum(e["type"] == "report" for e in edges),
               "problems": sum(d["level"] == "problem" for d in diagnostics),
               "infos": sum(d["level"] == "info" for d in diagnostics)}
    return {"ieds": ieds, "blocks": blocks, "edges": edges, "diagnostics": diagnostics, "summary": summary}
