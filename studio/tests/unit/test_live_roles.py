"""Real IEDs are observed, never simulated or commanded (LIVE-04/05/08)."""
import pytest
from studio_app.contracts.errors import StudioError
from studio_app.domain.network_settings import command_refusal, real_endpoint
from studio_app.projects.ied_settings import endpoint_conflicts, update_ied
from studio_app.projects.service import ProjectService
from studio_app.simulation.coordinator import Coordinator
from studio_app.runtime.event_bus import EventBus


def test_operating_real_ied_is_refused():
    assert "nothing was sent from here" in command_refusal({"name": "Relay", "role": "real"})
    assert "left out" in command_refusal({"name": "Relay", "role": "off"})
    assert command_refusal({"name": "Relay", "role": "simulated"}) is None


def test_real_ied_is_not_simulated(tmp_path):
    projects = ProjectService(tmp_path)
    simulated = projects.add_template("Sim")[0]
    real = projects.add_template("Real")[0]
    update_ied(projects, real["id"], {"role": "real", "live": {"ip": "10.9.9.9"}})
    started = []

    class FakeSupervisor:
        workers = {}

        def start(self, ieds, network):
            started.extend(ied["name"] for ied in ieds)

        def stop(self, ieds):
            pass

    coordinator = Coordinator(projects, EventBus(), tmp_path / "missing.exe", tmp_path)
    coordinator.supervisor = FakeSupervisor()
    coordinator.live.start = lambda project, selected=None: None
    try:
        coordinator.start()
        assert started == ["Sim"]
        assert real["id"] not in coordinator.sample_writer.locks and simulated["id"] in coordinator.sample_writer.locks
        assert (real["id"], real["signals"][0]["ref"]) in coordinator.index
    finally:
        coordinator.close()


def test_endpoint_prefill_and_override(tmp_path):
    ied = {"name": "R", "role": "real", "scl_address": {"ip": "10.0.0.3"},
           "live": {"ip": "", "port": 102, "access": "auto_reports", "poll_ms": 1000}}
    assert real_endpoint(ied) == ("10.0.0.3", 102, "scd")
    ied["live"]["ip"] = "10.0.0.33"
    assert real_endpoint(ied) == ("10.0.0.33", 102, "override")
    assert real_endpoint({**ied, "scl_address": None, "live": {**ied["live"], "ip": ""}})[2] == "missing"
    projects = ProjectService(tmp_path)
    one = projects.add_template("One")[0]
    two = projects.add_template("Two")[0]
    update_ied(projects, one["id"], {"role": "real", "live": {"ip": "10.0.0.5"}})
    with pytest.raises(StudioError, match="both use 10.0.0.5:102"):
        update_ied(projects, two["id"], {"role": "real", "live": {"ip": "10.0.0.5"}})
    assert endpoint_conflicts(projects.project["ieds"], projects.project["network"]) == []
