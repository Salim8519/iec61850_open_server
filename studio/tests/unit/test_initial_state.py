"""Initial equipment start positions: defaults, application, and persistence."""
from studio_app.contracts.project import EquipmentData
from studio_app.simulation.initial_state import apply_initial_states


def equipment(kind="CB", mode="direct", position="open", mapping=None):
    source = {"ied_id": "ied", "ref": "status", "report_ref": None}
    return {"id": "n1", "data": {"kind": kind, "initial_position": position,
            "state_mapping": mapping if mapping is not None else
            ({"mode": mode, "source": source} if mode else None)}}


class FakeWriter:
    def __init__(self, ieds=("ied",)):
        self.locks = {ied: object() for ied in ieds}
        self.calls = []

    def apply(self, ied_id, items, source, guard=None):
        self.calls.append((ied_id, items, source))
        return {ref: value for ref, value, _ in items}


class FakeCoordinator:
    def __init__(self, nodes, ieds=("ied",)):
        self.projects = type("P", (), {"project": {"diagram": {"nodes": nodes}}})()
        self.sample_writer = FakeWriter(ieds)


def test_switches_default_to_open_initial_position():
    for kind in ("CB", "DS", "ES"):
        assert EquipmentData(label="x", kind=kind).initial_position == "open"
    assert EquipmentData(label="x", kind="CT").initial_position == "open"


def test_initial_states_apply_saved_positions_per_mapping_mode():
    contacts = {"mode": "contacts_to_dbpos",
                "open": {"ied_id": "ied", "ref": "open", "report_ref": None},
                "closed": {"ied_id": "ied", "ref": "closed", "report_ref": None}}
    nodes = [equipment("CB", position="closed"), equipment("DS"),
             equipment("ES", mode="single_to_dbpos", position="closed"),
             equipment("ES", mapping=contacts), equipment("CT", position="closed")]
    coordinator = FakeCoordinator(nodes)
    apply_initial_states(coordinator)
    (ied, items, source), = coordinator.sample_writer.calls
    assert (ied, source) == ("ied", "initial_state")
    # one closed direct point, one open direct point, one closed single point,
    # and one open contacts pair
    assert items == [("status", 2, None), ("status", 1, None),
                     ("status", True, None), ("open", True, None), ("closed", False, None)]


def test_unmapped_or_stopped_ieds_are_skipped():
    nodes = [equipment("CB", mode=None), equipment("DS")]
    coordinator = FakeCoordinator(nodes, ieds=("other",))
    apply_initial_states(coordinator)
    assert coordinator.sample_writer.calls == []
