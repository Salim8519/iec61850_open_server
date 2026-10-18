"""What the worker is told about GOOSE, and stale-engine detection (GOOSE-05/06)."""
import pytest
from studio_app.runtime.worker_settings import check_capabilities, goose_request, start_request

GUID = "{11111111-1111-1111-1111-111111111111}"
PUBLISHER = {"name": "P", "ip": "127.50.0.2", "port": 102, "lab_ip": "192.168.50.21", "edition": 2,
             "goose_publish": True, "goose": {"publishes": [{"gocb_ref": "P/LLN0$GO$g"}]}}


def test_loopback_mode_never_publishes():
    loopback = {"mode": "loopback", "adapter_guid": GUID}
    assert goose_request(PUBLISHER, loopback) == {"publish": False, "adapter_guid": ""}
    request = start_request(PUBLISHER, loopback, [], "x.cfg")
    assert request["ip"] == "127.50.0.2" and request["goose"]["publish"] is False
    lab = {"mode": "lab", "adapter_guid": GUID}
    assert goose_request(PUBLISHER, lab) == {"publish": True, "adapter_guid": GUID}
    assert start_request(PUBLISHER, lab, [], "x.cfg")["ip"] == "192.168.50.21"
    assert goose_request({**PUBLISHER, "goose_publish": False}, lab)["publish"] is False
    assert goose_request({**PUBLISHER, "goose": {"publishes": []}}, lab)["publish"] is False


def test_stale_worker_is_rejected():
    current = ["versioned_updates", "atomic_control_feedback", "control_selection_events",
               "standard_controls", "goose_aware"]
    check_capabilities({"capabilities": current})
    with pytest.raises(ValueError, match="outdated"):
        check_capabilities({"capabilities": current[:-1]})
