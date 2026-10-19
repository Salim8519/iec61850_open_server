"""What the worker is told about GOOSE, and stale-engine detection (GOOSE-05/06, ADR-0073)."""
import pytest
from studio_app.runtime.worker_settings import check_capabilities, goose_request, start_request

GUID = "{11111111-1111-1111-1111-111111111111}"
PUBLISHER = {"name": "P", "ip": "127.50.0.2", "port": 102, "lab_ip": "192.168.50.21", "edition": 2,
             "goose_publish": True, "goose": {"publishes": [{"gocb_ref": "P/LLN0$GO$g"}]}}


def test_workers_never_get_an_adapter():
    """GOOSE-05: in both network modes a worker publishes only to the app's GOOSE engine."""
    bus = "udp:50123:025B12000001"
    for network in ({"mode": "loopback", "adapter_guid": GUID}, {"mode": "lab", "adapter_guid": GUID}):
        request = start_request(PUBLISHER, network, [], "x.cfg", bus)
        assert request["goose"] == {"publish": True, "interface": bus}
        assert GUID not in str(request["goose"])
    assert start_request(PUBLISHER, {"mode": "loopback"}, [], "x.cfg")["ip"] == "127.50.0.2"
    assert start_request(PUBLISHER, {"mode": "lab", "adapter_guid": GUID}, [], "x.cfg", bus)["ip"] == "192.168.50.21"
    assert goose_request(PUBLISHER, "") == {"publish": False, "interface": ""}
    assert goose_request({**PUBLISHER, "goose_publish": False}, bus)["publish"] is False
    assert goose_request({**PUBLISHER, "goose": {"publishes": []}}, bus)["publish"] is False


def test_stale_worker_is_rejected():
    current = ["versioned_updates", "atomic_control_feedback", "control_selection_events",
               "standard_controls", "goose_aware", "goose_bus"]
    check_capabilities({"capabilities": current})
    with pytest.raises(ValueError, match="outdated"):
        check_capabilities({"capabilities": current[:-1]})
