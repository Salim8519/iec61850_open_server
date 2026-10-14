"""Live, dead and earthed sections from switch positions (spec energization, POWER-01..04)."""
from studio_app.domain.energization import energize

OPEN, CLOSED, MIDDLE = 1, 2, 0


def node(node_id, kind, **data):
    return {"id": node_id, "position": {"x": 0, "y": 0}, "data": {"label": node_id, "kind": kind, "rotation": 0, **data}}


def wire(a, b):
    """Vertical wire: a's bottom terminal to b's top terminal."""
    return {"id": f"{a}-{b}", "source": a, "target": b, "sourceHandle": "south", "targetHandle": "north"}


def feeder(*extra_nodes, extra_wires=()):
    """Infeed -> bus junction -> DS -> CB -> CT -> line junction -> (extras)."""
    nodes = [node("feed", "INFEED", infeed_state="live"), node("bus", "JUNCTION"), node("ds", "DS"),
             node("cb", "CB"), node("ct", "CT"), node("line", "JUNCTION"), *extra_nodes]
    wires = [wire("feed", "bus"), wire("bus", "ds"), wire("ds", "cb"), wire("cb", "ct"), wire("ct", "line"), *extra_wires]
    return {"nodes": nodes, "edges": wires}


def test_switch_position_splits_sections():
    closed = energize(feeder(), {"ds": CLOSED, "cb": CLOSED})
    assert set(closed["edges"].values()) == {"live"}
    open_cb = energize(feeder(), {"ds": CLOSED, "cb": OPEN})["edges"]
    assert open_cb["ds-cb"] == "live" and open_cb["cb-ct"] == "dead" and open_cb["ct-line"] == "dead"
    assert energize(feeder(), {"ds": CLOSED, "cb": OPEN})["nodes"]["cb"] == ["live", "dead"]


def test_unknown_position_is_maybe_live():
    for unknown in ({"ds": CLOSED, "cb": MIDDLE}, {"ds": CLOSED}, {"ds": CLOSED, "cb": 3}):
        edges = energize(feeder(), unknown)["edges"]
        assert edges["ds-cb"] == "live"
        assert edges["cb-ct"] == "maybe_live" and edges["ct-line"] == "maybe_live"
    # Open further up beats an unknown switch below it: certainly dead.
    assert energize(feeder(), {"ds": OPEN})["edges"]["ct-line"] == "dead"


def test_earthing_and_earth_fault():
    diagram = feeder(node("es", "ES"), extra_wires=[wire("line", "es")])
    earthed = energize(diagram, {"ds": CLOSED, "cb": OPEN, "es": CLOSED})
    assert earthed["edges"]["ct-line"] == "earthed" and earthed["edges"]["ds-cb"] == "live"
    fault = energize(diagram, {"ds": CLOSED, "cb": CLOSED, "es": CLOSED})
    assert fault["edges"]["ct-line"] == "earth_fault" and fault["edges"]["feed-bus"] == "earth_fault"
    assert fault["nodes"]["es"] == ["earth_fault"]
    # An open earth switch earths nothing.
    assert energize(diagram, {"ds": CLOSED, "cb": OPEN, "es": OPEN})["edges"]["ct-line"] == "dead"


def test_dead_infeed_and_non_power_items():
    diagram = feeder(node("trip", "TRIP_RELAY"), node("vt", "VT"), node("vt2", "VT"),
                     extra_wires=[wire("cb", "trip"), wire("bus", "vt"), wire("vt", "vt2")])
    diagram["nodes"][0]["data"]["infeed_state"] = "dead"
    result = energize(diagram, {"ds": CLOSED, "cb": CLOSED})
    assert set(result["edges"].values()) == {"dead"}
    assert "cb-trip" not in result["edges"] and "trip" not in result["nodes"], "control items carry no state"
    diagram["nodes"][0]["data"]["infeed_state"] = "live"
    result = energize(diagram, {"ds": CLOSED, "cb": CLOSED})
    assert result["edges"]["bus-vt"] == "live" and result["edges"]["vt-vt2"] == "dead", "a VT never passes power on"
