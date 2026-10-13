"""Build the exact, fingerprinted list of Windows changes for the lab network."""
import hashlib
import json

GROUP = "Substation Studio"
RULE_PREFIX = "SubstationStudio-MMS-"


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


def describe(item):
    """One plain sentence a beginner can understand."""
    if item["kind"] == "alias":
        if item["action"] == "keep":
            return f"Use address {item['ip']} that is already on '{item['adapter_name']}' for {item['for']}"
        return f"Add address {item['ip']}/{item['prefix']} to '{item['adapter_name']}' for {item['for']}"
    if item["action"] == "keep":
        return "The Windows Firewall rule for Substation Studio already exists"
    return (f"Allow other devices to reach the simulated IEDs on TCP port "
            f"{', '.join(map(str, item['ports']))} through Windows Firewall")


def describe_removal(item):
    if item["kind"] == "alias":
        return f"Remove address {item['ip']} that Studio added to '{item['adapter_name']}'"
    return "Remove the Windows Firewall rule that Studio added for the simulated IEDs"


def apply_command(item):
    if item["kind"] == "alias":
        return (f"New-NetIPAddress -InterfaceIndex {int(item['adapter_index'])} -IPAddress {item['ip']} "
                f"-PrefixLength {int(item['prefix'])} -AddressFamily IPv4 -SkipAsSource $true "
                "-PolicyStore ActiveStore")
    return (f"New-NetFirewallRule -Name {_ps_string(item['name'])} -DisplayName "
            f"{_ps_string('Substation Studio simulated IEDs (MMS)')} -Group {_ps_string(item['group'])} "
            f"-Direction Inbound -Action Allow -Protocol TCP -LocalPort {','.join(map(str, item['ports']))} "
            f"-LocalAddress {','.join(item['addresses'])} -Program {_ps_string(item['program'])} -Profile Any")


def remove_command(item):
    if item["kind"] == "alias":
        return (f"Remove-NetIPAddress -InterfaceIndex {int(item['adapter_index'])} -IPAddress {item['ip']} "
                "-PolicyStore ActiveStore -Confirm:$false")
    return f"Remove-NetFirewallRule -Name {_ps_string(item['name'])}"
