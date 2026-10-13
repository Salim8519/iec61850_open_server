"""Exact Windows commands and ownership in the lab change plan (NET-06, NET-07)."""
from studio_app.networking.change_plan import apply_command, build_plan, remove_command

ADAPTER = {"guid": "{11111111-1111-1111-1111-111111111111}", "name": "Lab NIC", "index": 7,
           "addresses": [{"ip": "192.168.50.10", "prefix": 24}]}
PROGRAM = r"C:\Studio\native\studio_ied.exe"


def rows(*addresses):
    return [{"name": f"IED{i}", "proposed": ip, "prefix": 24,
             "status": "already_on_adapter" if ip == "192.168.50.10" else "ok"} for i, ip in enumerate(addresses)]


def test_alias_is_non_persistent_and_skip_as_source():
    plan = build_plan(ADAPTER, rows("192.168.50.21"), {102}, PROGRAM)
    alias = plan["items"][0]
    command = apply_command(alias)
    assert "-SkipAsSource $true" in command and "-PolicyStore ActiveStore" in command
    assert "-InterfaceIndex 7" in command and "-IPAddress 192.168.50.21" in command
    assert "-PolicyStore ActiveStore" in remove_command(alias)


def test_firewall_rule_is_narrow():
    plan = build_plan(ADAPTER, rows("192.168.50.21", "192.168.50.22"), {102}, PROGRAM)
    rule = plan["items"][-1]
    command = apply_command(rule)
    assert rule["kind"] == "firewall" and rule["group"] == "Substation Studio"
    assert "-Direction Inbound" in command and "-Protocol TCP" in command and "-LocalPort 102" in command
    assert "-LocalAddress 192.168.50.21,192.168.50.22" in command
    assert f"-Program '{PROGRAM}'" in command


def test_existing_address_is_not_owned():
    plan = build_plan(ADAPTER, rows("192.168.50.10", "192.168.50.21"), {102}, PROGRAM)
    existing, added = plan["items"][0], plan["items"][1]
    assert existing["owned"] is False and existing["action"] == "keep"
    assert added["owned"] is True and added["action"] == "add"
    again = build_plan(ADAPTER, rows("192.168.50.10", "192.168.50.21"), {102}, PROGRAM,
                       existing_rules=[plan["items"][-1]["name"]])
    assert again["items"][-1]["owned"] is False and again["fingerprint"] != plan["fingerprint"]


def test_fingerprint_changes_with_the_plan():
    first = build_plan(ADAPTER, rows("192.168.50.21"), {102}, PROGRAM)["fingerprint"]
    assert first == build_plan(ADAPTER, rows("192.168.50.21"), {102}, PROGRAM)["fingerprint"]
    assert first != build_plan(ADAPTER, rows("192.168.50.22"), {102}, PROGRAM)["fingerprint"]
