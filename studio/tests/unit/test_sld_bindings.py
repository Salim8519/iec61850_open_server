"""LNode-based signal and control binding for imported switches (spec SLD)."""
from studio_app.domain.sld_bindings import bind

DOMAINS = {("P1", "LD0"): "P1LD0"}


def ied(controls=(), status=("P1LD0/XCBR1.Pos.stVal",)):
    return {"id": "i1", "name": "P1",
            "signals": [{"ref": ref, "kind": "Dbpos", "writable": True, "fc": "ST"} for ref in status],
            "controls": [{"ref": ref, "cdc": "DPC"} for ref in controls]}


def lnode(cls, inst="1", name="P1"):
    return {"iedName": name, "ldInst": "LD0", "prefix": "", "lnClass": cls, "lnInst": inst}


def test_cswi_control_is_preferred():
    device = ied(controls=["P1LD0/CSWI1.Pos", "P1LD0/XCBR1.Pos"])
    state, control, reasons = bind("CB", [lnode("XCBR"), lnode("CSWI")], [device], DOMAINS)
    assert state["source"]["ref"] == "P1LD0/XCBR1.Pos.stVal"
    assert control["ref"] == "P1LD0/CSWI1.Pos" and reasons == []
    state, control, _ = bind("CB", [lnode("XCBR")], [device], DOMAINS)
    assert control["ref"] == "P1LD0/XCBR1.Pos"


def test_missing_ied_leaves_equipment_unbound():
    state, control, reasons = bind("CB", [lnode("XCBR", name="P9")], [ied()], DOMAINS)
    assert state is None and control is None
    assert any("P9" in reason and "not loaded" in reason for reason in reasons)
    state, control, reasons = bind("CB", [], [ied()], DOMAINS)
    assert (state, control) == (None, None) and reasons


def test_csw_is_never_guessed_from_naming():
    device = ied(controls=["P1LD0/CSWI1.Pos"])
    state, control, reasons = bind("CB", [lnode("XCBR")], [device], DOMAINS)
    assert state is not None and control is None
    assert any("Properties" in reason for reason in reasons)


def test_unassigned_lnode_is_ignored():
    state, control, reasons = bind("DS", [lnode("XSWI", name="None")], [ied()], DOMAINS)
    assert (state, control) == (None, None) and reasons
