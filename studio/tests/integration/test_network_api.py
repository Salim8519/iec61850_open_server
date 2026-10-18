"""Network HTTP workflow with Windows calls replaced by fakes (NET-02/05/10)."""
import json
from pathlib import Path

import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]
GUID = "{11111111-1111-1111-1111-111111111111}"
ADAPTER = {"guid": GUID, "name": "Lab NIC", "description": "USB Ethernet", "status": "Up", "index": 7,
           "mac": "", "speed": "1 Gbps", "kind": "ethernet", "default_route": False,
           "addresses": [{"ip": "192.168.50.10", "prefix": 24, "skip_as_source": False, "origin": "Manual"}]}


@pytest.fixture
def client(tmp_path):
    settings = Settings(tmp_path, ROOT / "studio/native/build/Release/studio_ied.exe", tmp_path)
    app = create_app(settings, "token")
    network = app.extensions["network"]
    network.calls = []
    network._list_adapters = lambda: [json.loads(json.dumps(ADAPTER))]
    network._list_rules = lambda: []
    network._probe = lambda addresses: set()

    def elevate(request, digest):
        network.calls.append(json.loads(Path(request).read_text()))
        items = network.calls[-1]["items"]
        from studio_app.networking.journal import item_key
        Path(request).with_suffix(".result.json").write_text(json.dumps(
            {"refused": False, "results": [{"key": item_key(item), "ok": True, "message": ""} for item in items]}))
    network._elevate = elevate
    test_client = app.test_client()
    test_client.post("/api/v1/session", json={"token": "token"})
    test_client.post("/api/v1/ieds/template", json={"name": "Bay"})
    yield app, test_client
    app.extensions["runtime"].close()


def test_admin_api_rejects_remote_callers(client):
    _, test_client = client
    response = test_client.get("/api/v1/network", environ_base={"REMOTE_ADDR": "192.168.50.77"})
    assert response.status_code == 403
    assert test_client.get("/api/v1/network").status_code == 200


def test_apply_requires_matching_fingerprint(client):
    app, test_client = client
    ied = app.extensions["projects"].project["ieds"][0]
    preview = test_client.post("/api/v1/network/plan/preview", json={
        "adapter_guid": GUID, "strategy": "adapter"}).json
    assert preview["ready"] and preview["needs_admin"]
    assert test_client.post("/api/v1/network/plan/apply", json={"fingerprint": preview["fingerprint"]}).status_code == 400
    stale = test_client.post("/api/v1/network/plan/apply", json={"fingerprint": "0" * 64, "confirm": True})
    assert stale.status_code == 409
    done = test_client.post("/api/v1/network/plan/apply", json={"fingerprint": preview["fingerprint"],
                                                                 "confirm": True}).json
    assert done["applied"] is True
    network = app.extensions["projects"].project["network"]
    assert network["mode"] == "lab" and network["adapter_guid"] == GUID
    assert ied["lab_ip"] == preview["rows"][0]["proposed"]
    kinds = [item["kind"] for item in app.extensions["network"].calls[0]["items"]]
    assert kinds == ["alias", "firewall"]
    assert len(app.extensions["network"].journal.outstanding()) == 2


def test_endpoint_override_keeps_scl(client):
    app, test_client = client
    ied = app.extensions["projects"].project["ieds"][0]
    source = ied["source"]
    response = test_client.put(f"/api/v1/ieds/{ied['id']}/settings", json={
        "role": "real", "live": {"ip": "10.20.30.40", "port": 102}})
    assert response.status_code == 200, response.json
    assert ied["live"]["ip"] == "10.20.30.40" and ied["role"] == "real"
    assert ied["source"] == source and "10.20.30.40" not in ied["source"]
    bad = test_client.put(f"/api/v1/ieds/{ied['id']}/settings", json={"live": {"ip": "224.0.0.1"}})
    assert bad.status_code == 400 and "normal device address" in bad.json["error"]
    lab = test_client.put(f"/api/v1/ieds/{ied['id']}/settings", json={"lab_ip": "127.0.0.9"})
    assert lab.status_code == 400 and "normal device address" in lab.json["error"]


def test_free_address_picker_skips_addresses_answering_on_the_network(client):
    """Change 0091: the VM picked .21, which another PC's lab IED already used, and only reported it."""
    app, test_client = client
    network = app.extensions["network"]
    probed = []

    def probe(addresses):
        probed.append(list(addresses))
        return {address for address in addresses if address in {"192.168.50.21", "192.168.50.22"}}
    network._probe = probe
    preview = test_client.post("/api/v1/network/plan/preview", json={"adapter_guid": GUID, "strategy": "adapter"}).json
    row = preview["rows"][0]
    assert row["proposed"] == "192.168.50.23" and row["status"] == "ok" and preview["ready"]
    assert len(probed) == 3, "picks again only while the picked address answers"
    ied = app.extensions["projects"].project["ieds"][0]
    typed = test_client.post("/api/v1/network/plan/preview", json={
        "adapter_guid": GUID, "strategy": "adapter", "manual": {ied["id"]: {"ip": "192.168.50.21", "prefix": 24}}}).json
    assert typed["rows"][0]["status"] == "in_use_on_network", "an address you typed is reported, never changed"
