"""Resolve which GOOSE blocks an IED subscribes to, from its ExtRef inputs."""
from .communication import read_communication
from .goose_controls import goose_controls


def _ln_name(element):
    if element.tag == "LN0":
        return "LLN0"
    return element.get("prefix", "") + element.get("lnClass", "") + element.get("inst", "")


def _datasets(ln0):
    return [{"name": item.get("name", ""), "members": [], "fcdas": item.findall("FCDA")}
            for item in ln0.findall("DataSet")]


def publisher_catalogue(root):
    """Every publishable GoCB in the document, keyed by (ied, ldInst, cbName)."""
    catalogue = {}
    for ied in root.findall("IED"):
        name = ied.get("name", "")
        communication, _ = read_communication(root, name)
        for ld in ied.findall("AccessPoint/Server/LDevice"):
            ln0 = ld.find("LN0")
            if ln0 is None:
                continue
            inst = ld.get("inst", "")
            domain = ld.get("ldName") or name + inst
            datasets = _datasets(ln0)
            for control in goose_controls(ln0, domain, inst, communication["gse"], datasets, name, []):
                fcdas = next(item["fcdas"] for item in datasets if item["name"] == control["dataset"])
                catalogue[(name, inst, control["name"])] = {
                    **{key: value for key, value in control.items() if key != "members"},
                    "publisher": name, "fcdas": [dict(item.attrib) for item in fcdas]}
    return catalogue


def _matches(extref, fcda):
    keys = ("ldInst", "prefix", "lnClass", "lnInst", "doName")
    if any((extref.get(key) or "") != (fcda.get(key) or "") for key in keys):
        return False
    return not fcda.get("daName") or not extref.get("daName") or (
        extref.get("daName") == fcda.get("daName") or extref["daName"].startswith(fcda["daName"] + "."))


def _member(values, fcdas):
    """Position of the input in the block's dataset and the path left below that member."""
    for index, fcda in enumerate(fcdas):
        if _matches(values, fcda):
            wanted, given = values.get("daName") or "", fcda.get("daName") or ""
            return index, wanted[len(given):].lstrip(".") if wanted.startswith(given) else ""
    return None, ""


def _lgos_targets(ied):
    targets = {}
    for ld in ied.findall("AccessPoint/Server/LDevice"):
        domain = ld.get("ldName") or ied.get("name", "") + ld.get("inst", "")
        for ln in ld.findall("LN"):
            if ln.get("lnClass") != "LGOS":
                continue
            value = ln.findtext("DOI[@name='GoCBRef']/DAI[@name='setSrcRef']/Val")
            if value and value.strip():
                targets[value.strip()] = f"{domain}/{_ln_name(ln)}"
    return targets


def goose_subscriptions(root, ied, catalogue=None):
    """Return subscriptions and diagnostics for one subscriber IED element."""
    catalogue = publisher_catalogue(root) if catalogue is None else catalogue
    lgos = _lgos_targets(ied)
    subscriptions, diagnostics = {}, []
    for ld in ied.findall("AccessPoint/Server/LDevice"):
        ld_domain = ld.get("ldName") or ied.get("name", "") + ld.get("inst", "")
        for ln in list(ld):
            for extref in ln.findall("Inputs/ExtRef"):
                values = dict(extref.attrib)
                if values.get("serviceType") not in (None, "GOOSE") or not values.get("iedName"):
                    continue
                source = f"{values['iedName']}/{values.get('ldInst', '')}"
                if values.get("srcCBName"):
                    key = (values["iedName"], values.get("srcLDInst") or values.get("ldInst", ""),
                           values["srcCBName"])
                    candidates = [key] if key in catalogue else []
                    if not candidates:
                        diagnostics.append(f"Input {values.get('intAddr') or values.get('doName')} expects "
                                           f"GOOSE block {key[0]}/{key[1]}/{key[2]}, which is not a "
                                           "publishable GOOSE block in this SCD")
                        continue
                else:
                    candidates = [key for key, control in catalogue.items() if key[0] == values["iedName"]
                                  and any(_matches(values, fcda) for fcda in control["fcdas"])]
                    if len(candidates) != 1:
                        if values.get("serviceType") == "GOOSE":
                            reason = "no" if not candidates else "more than one"
                            diagnostics.append(f"Input from {source} matches {reason} GOOSE block; "
                                               "add srcCBName to the ExtRef")
                        continue
                control = catalogue[candidates[0]]
                entry = subscriptions.setdefault(control["gocb_ref"], {
                    **{key: value for key, value in control.items() if key != "fcdas"},
                    "lgos": lgos.get(control["gocb_ref"]), "member_count": len(control["fcdas"]), "inputs": []})
                member, member_path = _member(values, control["fcdas"])
                entry["inputs"].append({"int_addr": values.get("intAddr", ""), "desc": values.get("desc", ""),
                                        "match": "named" if values.get("srcCBName") else "data",
                                        "ld": ld_domain, "member": member, "member_path": member_path,
                                        "source": f"{source}/{values.get('prefix', '')}"
                                                  f"{values.get('lnClass', '')}{values.get('lnInst', '')}."
                                                  f"{values.get('doName', '')}"
                                                  + (f".{values['daName']}" if values.get("daName") else "")})
    return list(subscriptions.values()), list(dict.fromkeys(diagnostics))
