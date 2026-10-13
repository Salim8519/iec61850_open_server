"""Propose one lab address per simulated IED and explain every conflict plainly."""
import ipaddress

STRATEGIES = {"scd", "adapter"}


def _adapter_networks(adapter):
    networks = []
    for item in adapter.get("addresses", []):
        interface = ipaddress.IPv4Interface(f"{item['ip']}/{item['prefix']}")
        if not interface.ip.is_link_local:
            networks.append(interface)
    return networks


def _scd_interface(ied):
    address = ied.get("scl_address") or {}
    if not address.get("ip"):
        return None
    mask = address.get("subnet") or "255.255.255.0"
    try:
        return ipaddress.IPv4Interface(f"{address['ip']}/{mask}")
    except ValueError:
        return None


def plan_addresses(ieds, adapter, all_adapters, strategy="scd"):
    """Return rows of {ied_id, name, scd_ip, proposed, prefix, status, hint}."""
    if strategy not in STRATEGIES:
        raise ValueError("Address strategy must be 'scd' or 'adapter'")
    own = {item["ip"] for item in adapter.get("addresses", [])}
    elsewhere = {item["ip"]: other["name"] for other in all_adapters if other["guid"] != adapter["guid"]
                 for item in other.get("addresses", [])}
    networks = _adapter_networks(adapter)
    taken = set(own) | set(elsewhere)
    rows, chosen = [], {}
    for ied in ieds:
        scd = _scd_interface(ied)
        row = {"ied_id": ied["id"], "name": ied["name"], "scd_ip": str(scd.ip) if scd else "",
               "proposed": "", "prefix": None, "status": "ok", "hint": ""}
        if strategy == "scd":
            if scd is None:
                row.update(status="missing_scd_address",
                           hint="This IED has no IP address in the SCD. Switch to 'free addresses on the "
                                "adapter' or type an address.")
            else:
                row.update(proposed=str(scd.ip), prefix=scd.network.prefixlen)
        else:
            network = next((n for n in networks if scd and scd.ip in n.network), networks[0] if networks else None)
            if network is None:
                row.update(status="adapter_has_no_network",
                           hint="This adapter has no usable IPv4 network yet. Use the SCD addresses instead.")
            elif scd and scd.ip in network.network and str(scd.ip) not in taken:
                row.update(proposed=str(scd.ip), prefix=network.network.prefixlen)
            else:
                free = next((str(host) for index, host in enumerate(network.network.hosts())
                             if index >= 20 and str(host) not in taken and str(host) not in chosen), "")
                row.update(proposed=free, prefix=network.network.prefixlen,
                           status="ok" if free else "network_full",
                           hint="" if free else "No free address left in this network.")
        _check(row, own, elsewhere, chosen)
        if row["proposed"]:
            chosen.setdefault(row["proposed"], row["name"])
        rows.append(row)
    return rows


def _check(row, own, elsewhere, chosen):
    address = row["proposed"]
    if not address or row["status"] != "ok":
        return
    if address in own:
        row.update(status="already_on_adapter", hint="This address is already on the adapter. Studio will "
                   "use it but will not remove it later, because Studio did not add it.")
    elif address in elsewhere:
        row.update(status="used_by_other_adapter",
                   hint=f"{address} already belongs to '{elsewhere[address]}'. Pick another address "
                        "or disconnect that adapter.")
    elif address in chosen:
        row.update(status="duplicate", hint=f"{chosen[address]} already uses {address}. Each IED needs "
                                            "its own address.")


def validate_manual(address, prefix):
    try:
        interface = ipaddress.IPv4Interface(f"{address}/{prefix}")
    except ValueError as error:
        raise ValueError("Enter an IPv4 address such as 192.168.1.21 and a prefix from 8 to 30") from error
    if not 8 <= interface.network.prefixlen <= 30 or interface.ip in (interface.network.network_address,
                                                                       interface.network.broadcast_address):
        raise ValueError("Use a host address inside a /8 to /30 network")
    if interface.ip.is_loopback or interface.ip.is_multicast or interface.ip.is_link_local:
        raise ValueError("Loopback, multicast and 169.254.x.x addresses cannot be used for the lab")
    return str(interface.ip), interface.network.prefixlen
