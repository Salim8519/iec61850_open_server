"""Bind imported switching equipment to IED signals named by SCL LNodes.

Only LNodes written on the equipment are used. Nothing is inferred from naming
conventions, so a missing binding is reported with a reason instead of guessed.
"""
SWITCH_CLASS = {"CB": "XCBR", "DS": "XSWI", "ES": "XSWI"}


def _reference(lnode, domains):
    domain = domains.get((lnode["iedName"], lnode["ldInst"]))
    if domain is None:
        return None
    return domain + "/" + lnode["prefix"] + lnode["lnClass"] + lnode["lnInst"] + ".Pos"


def _state(ied, ref):
    wanted = ref + ".stVal"
    signal = next((item for item in ied["signals"] if item["ref"] == wanted), None)
    if signal and signal["kind"] == "Dbpos" and signal["writable"] and signal["fc"] != "CF":
        return {"mode": "direct", "source": {"ied_id": ied["id"], "ref": wanted, "report_ref": None}}
    return None


def _control(ied, ref):
    control = next((item for item in ied["controls"] if item["ref"] == ref), None)
    if control and control.get("cdc") == "DPC":
        return {"ied_id": ied["id"], "ref": ref}
    return None


def bind(kind, lnodes, ieds, domains):
    """Return (state_mapping, control_binding, reasons) for one piece of equipment."""
    if kind not in SWITCH_CLASS:
        return None, None, []
    named = [item for item in lnodes if item["iedName"] and item["iedName"] != "None"]
    switch = next((item for item in named if item["lnClass"] == SWITCH_CLASS[kind]), None)
    command = next((item for item in named if item["lnClass"] == "CSWI"), None)
    if switch is None and command is None:
        return None, None, ["No IED is linked to this equipment in the SCD"]
    reasons, state, control = [], None, None
    by_name = {ied["name"]: ied for ied in ieds}
    for lnode in (item for item in (switch, command) if item):
        if lnode["iedName"] not in by_name:
            reasons.append(f"IED {lnode['iedName']} is in the SCD but not loaded in this project")
    if switch and switch["iedName"] in by_name:
        ref = _reference(switch, domains)
        state = _state(by_name[switch["iedName"]], ref) if ref else None
        if state is None:
            reasons.append(f"No position signal found at {ref or switch['lnClass']}")
    for lnode in (item for item in (command, switch) if item):
        if control is None and lnode["iedName"] in by_name:
            ref = _reference(lnode, domains)
            control = _control(by_name[lnode["iedName"]], ref) if ref else None
    if control is None and not reasons:
        reasons.append("No controllable switch command is linked; choose the control in Properties")
    elif control is None:
        reasons.append("No control bound; choose the control in Properties")
    return state, control, reasons
