"""Run-time endpoint checks for loopback and lab network modes (NET-03)."""
import pytest
from studio_app.contracts.errors import StudioError
from studio_app.networking.endpoints import preflight

LAB = {"mode": "lab", "adapter_guid": "{15A6105F-A715-444F-9D7C-AA259A2EED8A}", "adapter_label": "Lab NIC"}
LOOPBACK = {"mode": "loopback", "adapter_guid": "", "adapter_label": ""}


def ied(ip="127.50.0.240", port=18140, lab_ip=""):
    return {"name": "Relay", "ip": ip, "port": port, "lab_ip": lab_ip}


def test_loopback_mode_rejects_lan_address():
    with pytest.raises(StudioError, match="This PC only"):
        preflight([ied(ip="192.168.10.21")], LOOPBACK)
    preflight([ied()], LOOPBACK)


def test_lab_address_must_exist_on_adapter():
    with pytest.raises(StudioError, match="not set up on Lab NIC"):
        preflight([ied(lab_ip="192.168.77.21")], LAB, adapter_addresses={"192.168.77.99"})
    preflight([ied(lab_ip="")], LAB, adapter_addresses=set())


def test_duplicate_endpoints_are_rejected():
    with pytest.raises(StudioError, match="Two IEDs"):
        preflight([ied(), {**ied(), "name": "Other"}], LOOPBACK)
