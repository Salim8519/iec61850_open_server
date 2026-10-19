"""migrate_diagram vectors for archive formats 1 to 15, including malformed stored data."""

B = "1" * 32
P = "BayControllerCTRL/"
POS, LOC, ANIN = P + "XCBR1.Pos.stVal", P + "XCBR1.Loc.stVal", P + "GGIO1.AnIn1.mag.f"
ALL = ["BayController", "Bay2", "FEED1"]


def node(node_id, kind, **data):
    return {"id": node_id, "position": {"x": 0, "y": 0}, "data": {"label": node_id.upper(), "kind": kind, **data}}


def legacy_diagram():
    return {"nodes": [
        node("cb", "CB", ied_id=B, status_ref=POS, control_ref=P + "XCBR1.Pos", report_ref="", status_kind="Dbpos"),
        node("ds", "DS", ied_id=B, status_ref=LOC, control_ref=P + "GGIO1.SPCSO1", position_values=[1, 2]),
        node("es", "ES", ied_id=B, status_ref=ANIN, report_ref=P + "LLN0.RP.EventsRCB"),
        node("ct", "CT", ied_id=B, status_ref=ANIN, control_ref="zz"),
        node("vt", "VT", ied_id="9" * 32, status_ref=ANIN),
        node("relay", "RELAY", ied_id=B, status_ref=LOC),
        node("trip", "TRIP_RELAY", ied_id=B, status_ref=LOC, trip_polarity="operate_true"),
        node("trip2", "TRIP_RELAY", trip_polarity="operate_false"),
        node("trip3", "TRIP_RELAY"),
        {"id": "nolabel", "data": {"kind": "TRIP_RELAY"}},
        node("btn", "PUSH_BUTTON"), node("bus", "BUSBAR", width=300), node("load", "LOAD"),
        node("tr", "TRANSFORMER"), node("src", "SOURCE", width=5),
        node("cb2", "CB", control_binding={"ied_id": B, "ref": P + "XCBR1.Pos", "compatibility": "concen"}),
        node("cb3", "CB", control_binding={"ied_id": B, "ref": P + "XCBR1.Pos", "compatibility": "standard"}),
        node("relay2", "RELAY", state_mapping={"mode": "direct", "source": {"ied_id": B, "ref": LOC}}),
        node("relay3", "RELAY", state_mapping={"mode": "contacts_to_dbpos", "open": {}, "closed": {}}),
        node("relay4", "RELAY", state_mapping={"mode": "single_to_dbpos", "source": None}),
        node("relay5", "RELAY", state_mapping=None),
    ], "edges": [
        {"id": "w1", "source": "cb", "target": "btn"}, {"id": "w2", "source": "bus", "target": "cb"},
        {"id": "w3", "source": "cb", "target": "ds"}, {"id": "w4", "source": "btn", "target": "tr"},
        {"id": "w5", "source": "ds", "target": "es"},
    ], "extra": {"kept": True}}


MALFORMED = [None, [], "x", 5, {}, {"nodes": None}, {"nodes": 5}, {"nodes": "ab"}, {"nodes": {"a": 1}},
             {"nodes": [None]}, {"nodes": [5]}, {"nodes": [[1]]}, {"nodes": [{"data": None}]},
             {"nodes": [{"data": [1]}]}, {"nodes": [{"data": "x"}]}, {"nodes": [{}]},
             {"nodes": [{"data": {"control_binding": [1]}}]}, {"nodes": [{"data": {"control_binding": "x"}}]},
             {"nodes": [{"data": {"control_binding": 5}}]}, {"nodes": [{"data": {"control_binding": {"compatibility": 5}}}]},
             {"nodes": [{"data": {"kind": ["BUSBAR"]}}]}, {"nodes": [{"id": [1], "data": {"kind": "BUSBAR"}}]},
             {"nodes": [{"id": {"a": 1}, "data": {"kind": "CB"}}]},
             {"nodes": [{"id": 1, "data": {"kind": "BUSBAR"}}, {"id": 1.0, "data": {"kind": "CB"}}]},
             {"nodes": [{"id": "b", "data": {"kind": "BUSBAR"}}], "edges": [{"source": [1], "target": "x"}]},
             {"nodes": [{"id": "b", "data": {"kind": "BUSBAR"}}], "edges": [{"source": "b", "target": [1]}]},
             {"nodes": [{"id": "b", "data": {"kind": "PUSH_BUTTON"}}], "edges": [5]},
             {"nodes": [{"id": [1], "data": {"kind": "PUSH_BUTTON"}}]},
             {"nodes": [], "edges": None}, {"nodes": [], "edges": {"a": {}}},
             {"nodes": [{"id": "r", "data": {"kind": "RELAY", "state_mapping": [1]}}]},
             {"nodes": [{"id": "r", "data": {"kind": "RELAY", "state_mapping": {"mode": [1]}}}]},
             {"nodes": [{"id": "r", "data": {"kind": "RELAY", "state_mapping": "x"}}]},
             {"nodes": [{"id": "r", "data": {"kind": ["RELAY"], "ied_id": [1], "status_ref": "x"}}]},
             {"nodes": [{"id": "r", "data": {"kind": "CB", "ied_id": {"a": 1}, "status_ref": "x"}}]},
             {"nodes": [{"id": "r", "data": {"kind": ["CB"], "ied_id": B, "status_ref": POS}}]},
             {"nodes": [{"id": "r", "data": {"kind": ["CB"], "ied_id": B, "control_ref": P + "XCBR1.Pos"}}]},
             {"nodes": [{"id": "r", "data": {"kind": "CB", "ied_id": B, "control_ref": [1]}}]},
             {"nodes": [{"id": 5, "data": {"label": None, "kind": "TRIP_RELAY", "trip_polarity": "operate_true"}}]},
             {"nodes": [{"id": "t", "data": {"label": 1.5, "kind": "TRIP_RELAY", "trip_polarity": None}}]},
             {"nodes": [{"data": {"label": ["a", {"b": 'c"d'}], "kind": "BUSBAR"}}]}]


def cases(fixtures):
    out = []
    for version in range(1, 16):
        out.append({"name": f"legacy diagram v{version}", "fn": "migrate_diagram",
                    "args": {"diagram": legacy_diagram(), "version": version}, "ieds": ALL})
    for version in (1, 3, 5, 6, 8, 9, 11, 12, 15):
        for index, value in enumerate(MALFORMED):
            out.append({"name": f"malformed {index} v{version}", "fn": "migrate_diagram",
                        "args": {"diagram": value, "version": version}, "ieds": ALL})
    one_button = {"nodes": [node("b", "PUSH_BUTTON"), node("c", "CB")], "edges": [{"id": "w", "source": "b", "target": "c"}]}
    for version in (8, 9, 10):
        out.append({"name": f"one button wire v{version}", "fn": "migrate_diagram",
                    "args": {"diagram": one_button, "version": version}, "ieds": []})
    return out
