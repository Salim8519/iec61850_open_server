"""validate_diagram cross-reference vectors against the fixture IEDs."""

B = "1" * 32
B2 = "2" * 32
P = "BayControllerCTRL/"
POS, LOC, SPC, ANIN = P + "XCBR1.Pos.stVal", P + "XCBR1.Loc.stVal", P + "GGIO1.SPCSO1.stVal", P + "GGIO1.AnIn1.mag.f"
CF, HEALTH, QUALITY = P + "XCBR1.Pos.ctlModel", P + "LLN0.Health.stVal", P + "XCBR1.Pos.q"
REPORT, BUFFERED = P + "LLN0.RP.EventsRCB", P + "LLN0.BR.Buffered"
ALL = ["BayController", "Bay2", "FEED1"]


def sig(ref, ied=B, report=None):
    return {"ied_id": ied, "ref": ref, "report_ref": report}


def node(node_id, kind, **data):
    return {"id": node_id, "position": {"x": 0, "y": 0}, "data": {"label": node_id, "kind": kind, **data}}


def direct(ref, **kw):
    return {"mode": "direct", "source": sig(ref, **kw)}


def cases(fixtures):
    out = []

    def add(name, *nodes, edges=None, ieds=ALL):
        value = {"nodes": list(nodes)}
        if edges is not None:
            value["edges"] = edges
        out.append({"name": name, "fn": "validate_diagram", "args": {"diagram": value}, "ieds": ieds})

    add("duplicate nodes", node("a", "CB"), node("a", "DS"))
    add("duplicate wires", node("a", "CB"), node("b", "DS"),
        edges=[{"id": "w", "source": "a", "target": "b"}, {"id": "w", "source": "b", "target": "a"}])
    add("missing wire end", node("a", "CB"), edges=[{"id": "w", "source": "a", "target": "zz"}])
    add("good wires", node("a", "CB"), node("b", "DS"),
        edges=[{"id": "w", "source": "a", "target": "b", "sourceHandle": "s", "targetHandle": None}])
    add("relay statuses on CB", node("a", "CB", relay_statuses=[{"id": "s", "label": "x", "source": sig(LOC)}]))
    add("relay ok", node("a", "RELAY", relay_statuses=[{"id": "s1", "label": "Trip", "source": sig(LOC)},
                                                    {"id": "s2", "label": "Pos", "source": sig(POS)}]))
    add("relay duplicate", node("a", "RELAY", relay_statuses=[{"id": "s", "label": "x", "source": sig(LOC)}] * 2))
    add("relay unknown ied", node("a", "RELAY", relay_statuses=[{"id": "s", "label": "x", "source": sig(LOC, ied="9" * 32)}]))
    add("relay unknown signal", node("a", "RELAY", relay_statuses=[{"id": "s", "label": "x", "source": sig("nope")}]))
    add("relay report member", node("a", "RELAY", relay_statuses=[{"id": "s", "label": "x", "source": sig(POS, report=REPORT)}]))
    add("relay report not member", node("a", "RELAY", relay_statuses=[{"id": "s", "label": "x", "source": sig(LOC, report=REPORT)}]))
    add("relay report unknown", node("a", "RELAY", relay_statuses=[{"id": "s", "label": "x", "source": sig(POS, report="zz")}]))
    add("relay report empty", node("a", "RELAY", relay_statuses=[{"id": "s", "label": "x", "source": sig(POS, report="")}]))
    add("relay state mapping", node("a", "RELAY", state_mapping=direct(LOC)))
    add("infeed defaults", node("a", "INFEED"))
    add("infeed values", node("a", "INFEED", infeed_type="grid", infeed_state="dead"))
    add("infeed on CB", node("a", "CB", infeed_type="grid"))
    add("infeed state on CB", node("a", "CB", infeed_state="live"))
    add("infeed mapping", node("a", "INFEED", state_mapping=direct(LOC)))
    add("junction mapping", node("a", "JUNCTION", state_mapping=direct(LOC)))
    add("junction control", node("a", "JUNCTION", control_binding={"ied_id": B, "ref": P + "XCBR1.Pos"}))
    for kind in ("CB", "DS", "ES"):
        add(f"{kind} direct dbpos", node("a", kind, state_mapping=direct(POS)))
        add(f"{kind} direct bool", node("a", kind, state_mapping=direct(LOC)))
        add(f"{kind} single bool", node("a", kind, state_mapping={"mode": "single_to_dbpos", "source": sig(LOC)}))
        add(f"{kind} single dbpos", node("a", kind, state_mapping={"mode": "single_to_dbpos", "source": sig(POS)}))
        add(f"{kind} control ok", node("a", kind, control_binding={"ied_id": B, "ref": P + "XCBR1.Pos"}))
        add(f"{kind} control spc", node("a", kind, control_binding={"ied_id": B, "ref": P + "GGIO1.SPCSO1"}))
    contacts = {"mode": "contacts_to_dbpos", "open": sig(LOC), "closed": sig(SPC)}
    add("contacts ok", node("a", "CB", state_mapping=contacts))
    add("contacts same", node("a", "CB", state_mapping={**contacts, "closed": sig(LOC, report=REPORT)}))
    add("contacts open dbpos", node("a", "CB", state_mapping={**contacts, "open": sig(POS)}))
    add("contacts closed dbpos", node("a", "CB", state_mapping={**contacts, "closed": sig(POS)}))
    add("contacts on VT", node("a", "VT", state_mapping=contacts))
    add("CF signal", node("a", "CT", state_mapping=direct(CF)))
    add("quality signal", node("a", "CT", state_mapping=direct(QUALITY)))
    add("health signal", node("a", "CT", state_mapping=direct(HEALTH)))
    add("trip relay ok", node("a", "TRIP_RELAY", state_mapping=direct(LOC)))
    add("trip relay single", node("a", "TRIP_RELAY", state_mapping={"mode": "single_to_dbpos", "source": sig(LOC)}))
    add("trip relay dbpos", node("a", "TRIP_RELAY", state_mapping=direct(POS)))
    add("trip relay control", node("a", "TRIP_RELAY", control_binding={"ied_id": B, "ref": P + "GGIO1.SPCSO1"}))
    add("trip relay dpc", node("a", "TRIP_RELAY", control_binding={"ied_id": B, "ref": P + "XCBR1.Pos"}))
    for mode in ("spc_momentary", "latching_status", "self_resetting"):
        add(f"push {mode} state", node("a", "PUSH_BUTTON", push_mode=mode, state_mapping=direct(SPC)))
        add(f"push {mode} control", node("a", "PUSH_BUTTON", push_mode=mode,
                                          control_binding={"ied_id": B, "ref": P + "GGIO1.SPCSO1"}))
    add("push contacts", node("a", "PUSH_BUTTON", push_mode="latching_status", state_mapping=contacts))
    add("source ok", node("a", "SOURCE", state_mapping=direct(ANIN)))
    add("source st", node("a", "SOURCE", state_mapping=direct(HEALTH)))
    add("source single", node("a", "SOURCE", state_mapping={"mode": "single_to_dbpos", "source": sig(ANIN)}))
    add("source control", node("a", "SOURCE", control_binding={"ied_id": B, "ref": P + "XCBR1.Pos"}))
    add("CT single", node("a", "CT", state_mapping={"mode": "single_to_dbpos", "source": sig(LOC)}))
    add("CT direct ok", node("a", "CT", state_mapping=direct(ANIN)))
    add("unknown control ied", node("a", "CB", control_binding={"ied_id": "9" * 32, "ref": P + "XCBR1.Pos"}))
    add("unknown control ref", node("a", "CB", control_binding={"ied_id": B, "ref": "zz"}))
    add("Bay2 ok", node("a", "DS", state_mapping=direct("Bay2CTRL/XSWI1.Pos.stVal", ied=B2),
                        control_binding={"ied_id": B2, "ref": "Bay2CTRL/XSWI1.Pos"}))
    add("cross ied", node("a", "DS", state_mapping=direct(POS, ied=B2)))
    add("no ieds", node("a", "CB", state_mapping=direct(POS)), ieds=[])
    feed = fixtures["FEED1"]
    writable = [s for s in feed["signals"] if s["writable"] and s["fc"] != "CF"]
    for kind_name in sorted({s["kind"] for s in writable})[:12]:
        signal = next(s for s in writable if s["kind"] == kind_name)
        add(f"FEED1 VT {kind_name}", node("a", "VT", state_mapping=direct(signal["ref"], ied="3" * 32)))
        add(f"FEED1 SOURCE {kind_name}", node("a", "SOURCE", state_mapping=direct(signal["ref"], ied="3" * 32)))
    for control in feed["controls"]:
        add(f"FEED1 control {control['ref']}", node("a", "CB", control_binding={"ied_id": "3" * 32, "ref": control["ref"]}))
    for report in feed["reports"][:6]:
        member = (report.get("dataset_ref") or "")
        dataset = next((d for d in feed["datasets"] if d["ref"] == member), None)
        refs = dataset["signal_refs"] if dataset else []
        target = refs[0] if refs else writable[0]["ref"]
        add(f"FEED1 report {report['ref']}", node("a", "RELAY", relay_statuses=[
            {"id": "s", "label": "x", "source": sig(target, ied="3" * 32, report=report["ref"])}]))
    add("all kinds plain", *[node(f"n{i}", kind) for i, kind in enumerate(
        ["CB", "DS", "ES", "CT", "VT", "SOURCE", "RELAY", "TRIP_RELAY", "PUSH_BUTTON", "JUNCTION", "INFEED"])])
    return out
