"""Friendly switching hints from the drawing (spec interlock-hints, HINT-01..03)."""
from studio_app.domain.interlock_hints import hints_for

OPEN, CLOSED = 1, 2


def node(node_id, kind, **data):
    return {"id": node_id, "position": {"x": 0, "y": 0}, "data": {"label": node_id.upper(), "kind": kind, "rotation": 0, **data}}


def wire(a, b):
    return {"id": f"{a}-{b}", "source": a, "target": b, "sourceHandle": "south", "targetHandle": "north"}


def bay(*extra, wires=()):
    """Infeed -> bus -> Q1 (DS) -> tee -> Q0 (CB) -> CT -> line, with extras."""
    nodes = [node("feed", "INFEED"), node("bus", "JUNCTION"), node("q1", "DS"), node("tee", "JUNCTION"),
             node("q0", "CB"), node("ct", "CT"), node("line", "JUNCTION"), *extra]
    edges = [wire("feed", "bus"), wire("bus", "q1"), wire("q1", "tee"), wire("tee", "q0"), wire("q0", "ct"),
             wire("ct", "line"), *wires]
    return {"nodes": nodes, "edges": edges}


def test_disconnector_warns_about_closed_breaker():
    hints = hints_for(bay(), {"q1": CLOSED, "q0": CLOSED})
    for action in ("open", "close"):
        assert [hint["code"] for hint in hints["q1"][action]] == ["load"]
        assert "Open Q0 first" in hints["q1"][action][0]["text"] and hints["q1"][action][0]["related"] == ["q0"]
    assert "q1" not in hints_for(bay(), {"q1": CLOSED, "q0": OPEN}), "breaker open: nothing to say"
    unknown = hints_for(bay(), {"q1": CLOSED})["q1"]["open"]
    assert unknown[0]["code"] == "unknown" and "unknown" in unknown[0]["text"]


def test_earth_switch_on_a_powered_part():
    diagram = bay(node("q8", "ES"), wires=[wire("line", "q8")])
    live = hints_for(diagram, {"q1": CLOSED, "q0": CLOSED, "q8": OPEN})["q8"]
    assert [hint["code"] for hint in live["close"]] == ["earth"] and live["open"] == []
    assert "q8" not in hints_for(diagram, {"q1": CLOSED, "q0": OPEN, "q8": OPEN}), "dead part: earthing is fine"
    maybe = hints_for(diagram, {"q1": CLOSED, "q8": OPEN})["q8"]["close"]
    assert maybe and "may have power" in maybe[0]["text"]


def test_closing_onto_an_earthed_part():
    diagram = bay(node("q8", "ES"), wires=[wire("line", "q8")])
    hints = hints_for(diagram, {"q1": CLOSED, "q0": OPEN, "q8": CLOSED})
    assert [hint["code"] for hint in hints["q0"]["close"]] == ["earth"]
    assert hints["q0"]["open"] == []
    assert "Open the earth switch first" in hints["q0"]["close"][0]["text"]
    assert "q0" not in hints_for(diagram, {"q1": CLOSED, "q0": OPEN, "q8": OPEN})
