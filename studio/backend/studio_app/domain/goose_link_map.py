"""One GOOSE link map for the project: publisher -> control block -> subscribers.

Pure and deterministic. Identity is the control block reference (goCBRef), never the
APPID or MAC, because real files reuse both. Every link comes from an SCL declaration;
links whose block is never sent, or whose publisher is outside the project, are kept
with that status so the interface can say why nothing arrives.
"""
import hashlib

NOT_SENT = {"disabled_in_file": "the file switches that block off", "no_dataset": "that block has no dataset",
            "no_address": "that block has no network address (GSE) in the file",
            "not_goose": "that block is not a GOOSE block", "not_sendable": "that block cannot be sent as written"}


def _id(*parts):
    return hashlib.sha256("|".join(str(part) for part in parts).encode()).hexdigest()[:16]


def _member(fcda):
    node = fcda.get("prefix", "") + fcda.get("lnClass", "") + fcda.get("lnInst", "")
    path = ".".join(part for part in (fcda.get("doName", ""), fcda.get("daName", "")) if part)
    return f"{fcda.get('ldInst', '')}/{node}.{path} [{fcda.get('fc', '')}]"


def _member_ref(block, fcda):
    """Data reference of a dataset member, used to recognise a publisher value change."""
    domain = (block["dataset_ref"] or "").split("/", 1)[0]
    if fcda.get("ldInst", block["ld_inst"]) != block["ld_inst"] or not domain:
        domain = block["publisher"] + fcda.get("ldInst", "")
    node = fcda.get("prefix", "") + fcda.get("lnClass", "") + fcda.get("lnInst", "")
    path = ".".join(part for part in (fcda.get("doName", ""), fcda.get("daName", "")) if part)
    return f"{domain}/{node}.{path}"


def _block(block):
    return {**{key: block[key] for key in ("publisher", "ld_inst", "name", "gocb_ref", "go_id", "dataset",
                                           "dataset_ref", "conf_rev", "app_id", "mac", "status", "label")},
            "members": [_member(item) for item in block["fcdas"]],
            "member_refs": [_member_ref(block, item) for item in block["fcdas"]]}


def _goose_links(declarations, involved):
    links = []
    for subscriber, subscriptions in sorted(declarations["subscriptions"].items()):
        for item in subscriptions:
            if subscriber not in involved and item["publisher"] not in involved:
                continue
            matches = {entry["match"] for entry in item["inputs"]}
            status = "declared" if item["status"] == "sendable" else item["status"]
            links.append({"id": _id("goose", item["gocb_ref"], subscriber), "gocb_ref": item["gocb_ref"],
                          "publisher": item["publisher"], "subscriber": subscriber, "block": item["name"],
                          "status": status, "remote": False, "match": "named" if matches == {"named"} else "data",
                          "inputs": [{key: entry[key] for key in ("int_addr", "desc", "source")}
                                     for entry in item["inputs"]]})
    return links


def _remote_links(declarations, involved):
    grouped = {}
    for subscriber, inputs in sorted(declarations["remote"].items()):
        if subscriber not in involved:
            continue
        for entry in inputs:
            mac = entry["hint"]["mac"] if entry["hint"] else ""
            key = (subscriber, entry["publisher"], entry["ld_inst"], entry["cb_name"])
            link = grouped.setdefault(key, {
                "id": _id("remote", *key), "publisher": entry["publisher"], "subscriber": subscriber,
                "block": entry["cb_name"], "status": "remote", "remote": True, "mac": mac,
                "gocb_ref": f"{entry['publisher']}{entry['ld_inst']}/LLN0$GO${entry['cb_name']}"
                if entry["cb_name"] and entry["ld_inst"] else None,
                "match": "named" if entry["cb_name"] else "vendor_hint" if entry["hint"] else "data", "inputs": []})
            link["mac"] = link["mac"] or mac
            link["inputs"].append({key: entry[key] for key in ("int_addr", "desc", "source")})
    return list(grouped.values())


def _names(blocks, limit=6):
    names = [f"{block['publisher']}/{block['name']}" for block in blocks]
    return ", ".join(names[:limit]) + (f" and {len(names) - limit} more" if len(names) > limit else "")


def _address_diagnostics(blocks):
    sendable = [block for block in blocks if block["status"] == "sendable"]
    diagnostics = []
    zero = [block for block in sendable if block["app_id"] == 0]
    if zero:
        diagnostics.append({"level": "hint", "code": "app_id_zero", "ied": None, "gocb_ref": None,
                            "message": f"{len(zero)} GOOSE block(s) use APPID 0000: {_names(zero)}. "
                                       "ScadaBench tells them apart by their control block reference.",
                            "hint": "Give each block its own APPID in the configuration tool if a switch "
                                    "or receiver filters GOOSE by APPID."})
    for field, code, level, word in (("app_id", "duplicate_app_id", "warning", "APPID"),
                                     ("mac", "shared_mac", "hint", "multicast MAC")):
        groups = {}
        for block in sendable:
            if block[field] not in (None, "", 0):
                groups.setdefault(block[field], []).append(block)
        for value, members in sorted(groups.items(), key=lambda item: str(item[0])):
            if len(members) < 2:
                continue
            shown = f"{value:04X}" if field == "app_id" else value
            diagnostics.append({"level": level, "code": code, "ied": None, "gocb_ref": None,
                                "message": f"{_names(members)} share {word} {shown}.",
                                "hint": "Receivers still separate them by control block reference, but a "
                                        f"filter on the {word} lets both through. Give each its own {word}."})
    return diagnostics


def _link_diagnostics(links, declarations, involved):
    diagnostics = []
    for name in sorted(involved):
        for message in declarations["problems"].get(name, []):
            diagnostics.append({"level": "problem", "code": "unresolved_input", "ied": name, "gocb_ref": None,
                                "message": message, "hint": "Check iedName, ldInst and srcCBName of this input "
                                "in the configuration tool. ScadaBench never guesses a publisher."})
    for link in links:
        block = f" {link['block']}" if link["block"] else ""
        where = f"{link['subscriber']} expects GOOSE{block} from {link['publisher']}"
        if link["status"] in NOT_SENT:
            diagnostics.append({"level": "warning", "code": link["status"], "ied": link["subscriber"],
                                "gocb_ref": link["gocb_ref"], "message": f"{where}, but {NOT_SENT[link['status']]}, "
                                "so it is never sent.", "hint": "Normal for spare inputs. To use it, fix the "
                                "block in the publisher's configuration tool and export the file again."})
        elif link["status"] == "remote":
            mac = f" (MAC {link['mac']} from the vendor input address)" if link["mac"] else ""
            diagnostics.append({"level": "info", "code": "remote_publisher", "ied": link["subscriber"],
                                "gocb_ref": link["gocb_ref"], "message": f"{where}{mac}, which is in none of this "
                                "project's files.", "hint": "Import that IED's file, or add it as a real device, "
                                "to supervise it. It is shown at the edge of the diagram."})
    return diagnostics


def build_link_map(declarations, project_ieds):
    """{ieds, blocks, links, diagnostics, summary} limited to links touching a project IED."""
    project = {ied["name"]: ied for ied in project_ieds}
    involved = set(project)
    links = sorted(_goose_links(declarations, involved) + _remote_links(declarations, involved),
                   key=lambda link: (link["publisher"], link["block"] or "", link["subscriber"]))
    named = set(involved) | {link["publisher"] for link in links} | {link["subscriber"] for link in links}
    blocks = [_block(block) for key, block in sorted(declarations["blocks"].items()) if key[0] in named]
    ieds = []
    for name in sorted(named):
        known = declarations["ieds"].get(name)
        ied = project.get(name, {})
        ieds.append({"name": name, "ied_id": ied.get("id"), "role": ied.get("role", "simulated") if ied else None,
                     "in_project": name in project, "remote": known is None,
                     "desc": (known or {}).get("desc", ""), "manufacturer": (known or {}).get("manufacturer", "")})
    diagnostics = _link_diagnostics(links, declarations, involved) + _address_diagnostics(blocks)
    return {"ieds": ieds, "blocks": blocks, "links": links, "diagnostics": diagnostics}
