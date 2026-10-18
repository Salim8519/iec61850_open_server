"""Windows Firewall for the station gateway on "All network adapters" (ADR-0062, change 0089).
Found between two machines: the gateway said "Listening on 0.0.0.0:2404" while Windows blocked
every SCADA, because a dismissed "allow access?" question had left Block rules."""
import hashlib
import json

import pytest
from studio_app.contracts.errors import StudioError
from studio_app.networking.change_plan import apply_command, describe, gateway_plan, remove_command
from studio_app.networking.gateway_firewall import GatewayFirewall
from studio_app.networking.helper.validation import HelperRefused, validate_request
from studio_app.networking.journal import NetworkJournal, item_key

IED = r"C:\Studio\native\studio_ied.exe"
GATEWAY = r"C:\Studio\native\studio_gateway.exe"
BLOCK = {"name": r"TCP Query User{CD50883C-152F-4A3A-864C-00A52154808F}c:\studio\native\studio_gateway.exe",
         "display": "studio_gateway.exe"}


def payload(items, action="apply"):
    data = json.dumps({"action": action, "fingerprint": "f", "items": items}).encode()
    return data, hashlib.sha256(data).hexdigest()


def test_plan_allows_the_gateway_port_and_switches_off_block_rules():
    plan = gateway_plan(GATEWAY, [], [BLOCK])
    allow, unblock = plan["items"]
    assert allow["kind"] == "gateway_firewall" and allow["action"] == "add"
    command = apply_command(allow)  # one approval covers IEC 104, DNP3 and Modbus, like Windows' own "Allow"
    assert "New-NetFirewallRule" in command and "-Protocol TCP" in command and GATEWAY in command
    assert "-LocalPort" not in command
    assert unblock["kind"] == "unblock" and "Disable-NetFirewallRule" in apply_command(unblock)
    assert "Enable-NetFirewallRule" in remove_command(unblock), "clean-up restores the rule, never deletes it"
    assert "Action -ne 'Block'" in apply_command(unblock), "the helper re-checks the live rule first"
    restore = remove_command(unblock)
    assert "Action" not in restore and "Direction -ne 'Inbound'" in restore and GATEWAY in restore,         "clean-up re-enables the rule even after Windows changed Block to Allow"
    assert "Remove-NetFirewallRule" in remove_command(allow)
    kept = gateway_plan(GATEWAY, [allow["name"]], [])
    assert [item["action"] for item in kept["items"]] == ["keep"] and "already exists" in describe(kept["items"][0])
    assert gateway_plan(GATEWAY.upper())["items"][0]["name"] == allow["name"], "Windows paths ignore case"


def test_helper_accepts_only_the_gateway_worker_and_plain_rule_names():
    allow, unblock = gateway_plan(GATEWAY, [], [BLOCK])["items"]
    data, digest = payload([allow, unblock])
    assert len(validate_request(data, digest, IED)["items"]) == 2
    for bad in ({**allow, "program": r"C:\Windows\System32\svchost.exe"}, {**allow, "name": "Something-else"}, {**unblock, "name": "x'; Remove-NetFirewallRule -Name *; '"},
                {**unblock, "program": IED}):
        data, digest = payload([bad])
        with pytest.raises(HelperRefused):
            validate_request(data, digest, IED)


def test_cleanup_finds_the_gateway_changes_in_the_journal(tmp_path):
    allow, unblock = gateway_plan(GATEWAY, [], [BLOCK])["items"]
    journal = NetworkJournal(tmp_path)
    journal.append("applied", allow)
    journal.append("applied", unblock)
    assert {item_key(item) for item in journal.outstanding()} == {f"firewall:{allow['name']}", f"unblock:{BLOCK['name']}"}
    journal.append("removed", unblock)
    assert [item["kind"] for item in journal.leftovers(lambda item: True)] == ["gateway_firewall"]


class Network:
    def __init__(self):
        self.sent = []

    def _run_helper(self, action, items, fingerprint):
        self.sent.append((action, [item["kind"] for item in items]))
        return [{"ok": True, "description": describe(item)} for item in items]


def test_preview_then_apply_sends_only_what_was_reviewed():
    network, blocking = Network(), [BLOCK]
    firewall = GatewayFirewall(network, GATEWAY, rules=lambda: [], blocking=lambda program: blocking)
    preview = firewall.preview()
    assert preview["open"] is False and preview["blocked"] is True and preview["needs_admin"] is True
    with pytest.raises(StudioError, match="confirm"):
        firewall.apply(preview["fingerprint"], confirm=False)
    blocking = []  # Windows changed after the review
    with pytest.raises(StudioError, match="changed since you reviewed"):
        firewall.apply(preview["fingerprint"], confirm=True)
    fresh = firewall.preview()
    assert firewall.apply(fresh["fingerprint"], confirm=True)["applied"] is True
    assert network.sent == [("apply", ["gateway_firewall"])]


def test_cleanup_reads_windows_firewall_once(tmp_path):
    """Change 0089: one PowerShell per journaled item made the clean-up list take ~10 s to appear."""
    from studio_app.networking.service import NetworkService
    first, unblock = gateway_plan(GATEWAY, [], [BLOCK])["items"]
    second = gateway_plan(GATEWAY.replace("Studio", "Other"))["items"][0]
    calls = []

    def firewall(names):
        calls.append(list(names))
        return {first["name"], second["name"]}, {BLOCK["name"]}
    service = NetworkService(None, tmp_path, IED, adapters=lambda: [], rules=lambda: [], firewall=firewall)
    for item in (first, unblock, second):
        service.journal.append("applied", item)
    preview = service.cleanup_preview()
    assert [item["kind"] for item in preview["items"]] == ["gateway_firewall", "unblock", "gateway_firewall"]
    assert calls == [[BLOCK["name"]]], "one read for every firewall item"
