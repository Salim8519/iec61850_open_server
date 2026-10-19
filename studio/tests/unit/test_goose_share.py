"""Sharing GOOSE with the network needs confirm and the adapter the user saw (GOOSE-19, ADR-0073)."""
import pytest
from studio_app.acquisition.goose_share import GooseShare
from studio_app.contracts.errors import StudioError
from studio_app.projects.service import ProjectService

CABLE = "{7B29C0B1-72E8-4C7D-9FAA-6A695893211D}"
ADAPTERS = [{"guid": CABLE, "name": "Ethernet", "kind": "ethernet", "status": "Up", "default_route": True}]


class Engine:
    def __init__(self, running=False):
        self.running = running
        self.calls = []

    def snapshot(self):
        return {"running": self.running, "uplink": "open", "share": "off", "conflicts": []}

    def stats(self):
        return {"delivered_wire": 0}

    def uplink(self, guid, share):
        self.calls.append((guid, share))

    def gate(self, ied_id, choice):
        self.calls.append((ied_id, choice))
        return {"ied_id": ied_id, "gate": choice}


class Manager:
    def __init__(self, running=False):
        self.engine = Engine(running)


def share(installed=True, running=False, adapters=ADAPTERS):
    return GooseShare(Manager(running), adapters=lambda: adapters,
                      npcap=lambda: {"installed": installed, "message": "Install Npcap"})


@pytest.fixture
def projects(tmp_path):
    service = ProjectService(tmp_path)
    service.add_template("Bay")
    return service


def test_sharing_is_refused_without_confirm_npcap_or_the_seen_adapter(projects):
    with pytest.raises(StudioError, match="confirm"):
        share().apply(projects, {"reach": "outside", "adapter_guid": CABLE})
    with pytest.raises(StudioError, match="Npcap"):
        share(installed=False).apply(projects, {"reach": "outside", "adapter_guid": CABLE, "confirm": True})
    with pytest.raises(StudioError, match="adapter changed"):
        share().apply(projects, {"reach": "outside", "adapter_guid": "{11111111-1111-1111-1111-111111111111}",
                                 "confirm": True})
    with pytest.raises(StudioError, match="No connected wired"):
        share(adapters=[]).apply(projects, {"reach": "outside", "adapter_guid": "", "confirm": True})
    assert projects.project["network"]["goose_reach"] == "inside"


def test_confirmed_sharing_is_saved_with_its_adapter_and_applied_while_running(projects):
    service = share(running=True)
    preview = service.apply(projects, {"reach": "outside", "adapter_guid": CABLE.lower(), "confirm": True})
    network = projects.project["network"]
    assert (network["goose_reach"], network["goose_share_guid"]) == ("outside", CABLE)
    assert service.manager.engine.calls[-1] == (CABLE, True)
    assert preview["reach"] == "outside" and preview["uplink"]["guid"] == CABLE
    service.apply(projects, {"reach": "inside"})
    assert projects.project["network"]["goose_share_guid"] == ""
    assert service.manager.engine.calls[-1] == (CABLE, False)  # still listens, never sends


def test_run_without_npcap_stays_inside_and_never_opens_an_adapter(projects):
    projects.project["network"].update(goose_reach="outside", goose_share_guid=CABLE)
    service = share(installed=False, running=True)
    service.start(projects.project)
    assert service.manager.engine.calls == [("", False)]
    assert "Npcap" in service.current["hold"]


def test_continue_after_a_conflict_needs_confirm(projects):
    service = share(running=True)
    with pytest.raises(StudioError, match="Confirm"):
        service.choose({"ied_id": "IED1", "choice": "continue"})
    with pytest.raises(StudioError):
        service.choose({"ied_id": "IED1", "choice": "maybe"})
    assert service.choose({"ied_id": "IED1", "choice": "stop"})["gate"] == "stop"
    assert service.choose({"ied_id": "IED1", "choice": "continue", "confirm": True})["gate"] == "continue"
