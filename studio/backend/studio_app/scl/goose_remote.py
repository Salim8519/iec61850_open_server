"""GOOSE inputs whose publisher IED is in none of the project's SCL files (remote publishers)."""
from .goose_vendor import packed_input_hint


def _source(values):
    path = f"{values.get('prefix', '')}{values.get('lnClass', '')}{values.get('lnInst', '')}.{values.get('doName', '')}"
    if values.get("daName"):
        path += f".{values['daName']}"
    return f"{values['iedName']}/{values.get('ldInst', '')}/{path}"


def remote_inputs(ied, known_names):
    """ExtRefs naming an IED outside `known_names` that are clearly GOOSE.

    An input counts only when it says so: serviceType GOOSE, a srcCBName, or a packed
    vendor intAddr that names a multicast MAC. Bare spare ExtRefs (intAddr + desc
    only) carry no iedName and are never listed.
    """
    found = []
    for extref in ied.iter("ExtRef"):
        values = dict(extref.attrib)
        publisher = values.get("iedName")
        if not publisher or publisher in known_names or values.get("serviceType") not in (None, "GOOSE"):
            continue
        hint = packed_input_hint(values.get("intAddr"))
        if hint and hint["publisher"] != publisher:
            hint = None
        if values.get("serviceType") != "GOOSE" and not values.get("srcCBName") and not hint:
            continue
        found.append({"publisher": publisher, "ld_inst": values.get("srcLDInst") or values.get("ldInst", ""),
                      "cb_name": values.get("srcCBName", ""), "int_addr": values.get("intAddr", ""),
                      "desc": values.get("desc", ""), "source": _source(values), "hint": hint})
    return found
