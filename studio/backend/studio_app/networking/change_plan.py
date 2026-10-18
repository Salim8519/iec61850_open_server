"""Build the exact, fingerprinted list of Windows changes for the lab network."""
import hashlib
import json

GROUP = "Substation Studio"  # kept: rules already created under this group must still be found and removed
RULE_PREFIX = "SubstationStudio-MMS-"
GATEWAY_GROUP = "ScadaBench"
GATEWAY_PREFIX = "ScadaBench-Gateway-"


def _ps_string(value):
    return "'" + str(value).replace("'", "''") + "'"


def build_plan(adapter, rows, ports, program, existing_rules=()):
    """Return {items, fingerprint}; already-present things are listed but not owned."""
    own = {item["ip"] for item in adapter.get("addresses", [])}
    items, addresses = [], []
    for row in rows:
        if not row.get("proposed") or row["status"] not in {"ok", "already_on_adapter"}:
            continue
        addresses.append(row["proposed"])
        present = row["proposed"] in own
        items.append({"kind": "alias", "adapter_guid": adapter["guid"], "adapter_index": adapter["index"],
                      "adapter_name": adapter["name"], "ip": row["proposed"], "prefix": row["prefix"],
                      "for": row["name"], "owned": not present, "action": "keep" if present else "add"})
    if addresses:
        ports = sorted({int(port) for port in ports})
        key = json.dumps([sorted(addresses), ports, str(program)], sort_keys=True)
        name = RULE_PREFIX + hashlib.sha256(key.encode()).hexdigest()[:12]
        present = name in set(existing_rules)
        items.append({"kind": "firewall", "name": name, "group": GROUP, "addresses": sorted(addresses),
                      "ports": ports, "program": str(program), "owned": not present,
                      "action": "keep" if present else "add"})
    canonical = json.dumps({"adapter": adapter["guid"], "items": items}, sort_keys=True)
    return {"items": items, "fingerprint": hashlib.sha256(canonical.encode()).hexdigest()}


def gateway_plan(program, existing_rules=(), blocking=()):
    """Let other computers reach the station gateway (ADR-0062): one allow rule for the gateway
    worker on TCP (like Windows' own "Allow" answer, so IEC 104, DNP3 and Modbus need one
    approval), and the Windows Block rules for that worker switched off (they are made when the
    "allow access?" question is dismissed). Block rules are disabled, never deleted, and the
    clean-up switches them on again."""
    name = GATEWAY_PREFIX + hashlib.sha256(str(program).lower().encode()).hexdigest()[:12]
    present = name in set(existing_rules)
    items = [{"kind": "gateway_firewall", "name": name, "group": GATEWAY_GROUP, "program": str(program),
              "owned": not present, "action": "keep" if present else "add"}]
    items += [{"kind": "unblock", "name": rule["name"], "display": rule.get("display", ""), "program": str(program),
               "owned": True, "action": "add"} for rule in blocking]
    canonical = json.dumps({"gateway": items}, sort_keys=True)
    return {"items": items, "fingerprint": hashlib.sha256(canonical.encode()).hexdigest()}


def describe(item):
    """One plain sentence a beginner can understand."""
    if item["kind"] == "gateway_firewall":
        if item["action"] == "keep":
            return "The Windows Firewall rule for the ScadaBench gateway already exists"
        return "Allow other computers to reach the ScadaBench gateway (IEC 104, DNP3 or Modbus TCP) through Windows Firewall"
    if item["kind"] == "unblock":
        return ("Switch off a Windows Firewall rule that blocks the ScadaBench gateway (Windows made it when its "
                "'allow access?' question was closed); clean-up switches it on again")
    if item["kind"] == "alias":
        if item["action"] == "keep":
            return f"Use address {item['ip']} that is already on '{item['adapter_name']}' for {item['for']}"
        return f"Add address {item['ip']}/{item['prefix']} to '{item['adapter_name']}' for {item['for']}"
    if item["action"] == "keep":
        return "The Windows Firewall rule for ScadaBench already exists"
    return (f"Allow other devices to reach the simulated IEDs on TCP port "
            f"{', '.join(map(str, item['ports']))} through Windows Firewall")


def describe_removal(item):
    if item["kind"] == "gateway_firewall":
        return "Remove the Windows Firewall rule that ScadaBench added for its gateway"
    if item["kind"] == "unblock":
        return "Switch the Windows Firewall rule that blocked the gateway on again, as it was before"
    if item["kind"] == "alias":
        return f"Remove address {item['ip']} that ScadaBench added to '{item['adapter_name']}'"
    return "Remove the Windows Firewall rule that ScadaBench added for the simulated IEDs"


def _blocking_check(item):
    """The helper re-checks, at the moment of the change, that the rule still blocks exactly the
    gateway worker; anything else is refused instead of being switched off."""
    return (f"$rule = Get-NetFirewallRule -Name {_ps_string(item['name'])}; "
            "if ($rule.Action -ne 'Block' -or $rule.Direction -ne 'Inbound' -or "
            f"(($rule | Get-NetFirewallApplicationFilter).Program -ne {_ps_string(item['program'])})) "
            "{ throw 'This rule is not a Windows block rule for the ScadaBench gateway; nothing was changed' }; ")


def _restore_check(item):
    """Clean-up only switches the rule back on, so its action does not matter: Windows rewrites it
    when the user answers its own "allow access?" question (change 0089: Block became Allow and the
    Block-only check refused to restore it). It must still be the inbound rule for the gateway."""
    return (f"$rule = Get-NetFirewallRule -Name {_ps_string(item['name'])}; "
            "if ($rule.Direction -ne 'Inbound' -or "
            f"(($rule | Get-NetFirewallApplicationFilter).Program -ne {_ps_string(item['program'])})) "
            "{ throw 'This rule no longer belongs to the ScadaBench gateway; nothing was changed' }; ")


def apply_command(item):
    if item["kind"] == "gateway_firewall":
        return (f"New-NetFirewallRule -Name {_ps_string(item['name'])} -DisplayName "
                f"{_ps_string('ScadaBench station gateway')} -Group {_ps_string(item['group'])} -Direction Inbound "
                f"-Action Allow -Protocol TCP -Program {_ps_string(item['program'])} -Profile Any")
    if item["kind"] == "unblock":
        return _blocking_check(item) + f"Disable-NetFirewallRule -Name {_ps_string(item['name'])}"
    if item["kind"] == "alias":
        return (f"New-NetIPAddress -InterfaceIndex {int(item['adapter_index'])} -IPAddress {item['ip']} "
                f"-PrefixLength {int(item['prefix'])} -AddressFamily IPv4 -SkipAsSource $true "
                "-PolicyStore ActiveStore")
    return (f"New-NetFirewallRule -Name {_ps_string(item['name'])} -DisplayName "
            f"{_ps_string('ScadaBench simulated IEDs (MMS)')} -Group {_ps_string(item['group'])} "
            f"-Direction Inbound -Action Allow -Protocol TCP -LocalPort {','.join(map(str, item['ports']))} "
            f"-LocalAddress {','.join(item['addresses'])} -Program {_ps_string(item['program'])} -Profile Any")


def remove_command(item):
    if item["kind"] == "unblock":
        return _restore_check(item) + f"Enable-NetFirewallRule -Name {_ps_string(item['name'])}"
    if item["kind"] == "alias":
        return (f"Remove-NetIPAddress -InterfaceIndex {int(item['adapter_index'])} -IPAddress {item['ip']} "
                "-PolicyStore ActiveStore -Confirm:$false")
    return f"Remove-NetFirewallRule -Name {_ps_string(item['name'])}"
