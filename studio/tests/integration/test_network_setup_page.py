"""Change 0098: the one-page network setup gets plain answers from the planner."""
import ipaddress
import json
from pathlib import Path

import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]
GUID = "{22222222-2222-2222-2222-222222222222}"


@pytest.fixture
def setup(tmp_path):
    app = create_app(Settings(tmp_path, ROOT / "studio/native/build/Release/studio_ied.exe", tmp_path), "token")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "token"})
    client.post("/api/v1/ieds/template", json={"name": "Bay"})
    ied = app.extensions["projects"].project["ieds"][0]
    scd_ip = (ied.get("scl_address") or {}).get("ip") or "192.168.60.20"
    own = str(ipaddress.IPv4Interface(f"{scd_ip}/24").network.network_address + 10)
    network = app.extensions["network"]
    network._list_adapters = lambda: [{"guid": GUID, "name": "Lab NIC", "description": "USB Ethernet",
                                       "status": "Up", "index": 7, "mac": "", "speed": "1 Gbps", "kind": "ethernet",
                                       "default_route": False, "addresses": [{"ip": own, "prefix": 24,
                                                                              "skip_as_source": False,
                                                                              "origin": "Manual"}]}]
    network._list_rules = lambda: []
    network._probe = lambda addresses: set()
    yield app, client, ied, scd_ip
    app.extensions["runtime"].close()


def test_nothing_to_publish_says_which_mode_to_choose(setup):
    _, client, ied, _ = setup
    client.put(f"/api/v1/ieds/{ied['id']}/settings", json={"role": "real"})
    answer = client.post("/api/v1/network/plan/preview", json={"adapter_guid": GUID, "strategy": "scd"})
    assert answer.status_code == 400
    assert "Be this IED" in json.dumps(answer.json)


def test_unusable_scd_address_offers_one_free_address(setup):
    """A taken SCD address, or none at all, gets one free address on the card's own network."""
    app, client, ied, scd_ip = setup
    has_scd = bool((ied.get("scl_address") or {}).get("ip"))
    app.extensions["network"]._probe = lambda addresses: {scd_ip} & set(addresses)
    preview = client.post("/api/v1/network/plan/preview", json={"adapter_guid": GUID, "strategy": "scd"}).json
    row = preview["rows"][0]
    assert row["status"] == ("in_use_on_network" if has_scd else "missing_scd_address") and not preview["ready"]
    alternative = row["alternative"]["ip"]
    assert alternative != scd_ip and ipaddress.IPv4Address(alternative) in ipaddress.IPv4Network(f"{scd_ip}/24", False)


def test_free_scd_address_offers_no_alternative(setup):
    _, client, ied, _ = setup
    preview = client.post("/api/v1/network/plan/preview", json={"adapter_guid": GUID, "strategy": "scd"}).json
    assert all("alternative" not in row for row in preview["rows"] if row["status"] == "ok")
