"""Read the SCL Communication section: IED addresses and GOOSE (GSE) addresses."""
import ipaddress
import re

MAC = re.compile(r"^([0-9A-Fa-f]{2}-){5}[0-9A-Fa-f]{2}$")


def _parameters(address):
    if address is None:
        return {}
    return {item.get("type"): (item.text or "").strip() for item in address.findall("P")}


def _ip(value):
    try:
        return str(ipaddress.IPv4Address(value)) if value else ""
    except ValueError:
        return ""


def _milliseconds(element):
    if element is None or not (element.text or "").strip():
        return None
    try:
        value = float(element.text.strip())
    except ValueError:
        return None
    # The SCL schema fixes unit="s" and multiplier="m", so the value is milliseconds.
    return int(round(value)) if value >= 0 else None


def _gse(element, warnings, ied_name):
    values = _parameters(element.find("Address"))
    where = f"GOOSE address {ied_name}/{element.get('ldInst')}/{element.get('cbName')}"
    mac = values.get("MAC-Address", "")
    try:
        app_id = int(values.get("APPID", ""), 16)
        vlan_id = int(values.get("VLAN-ID", "0") or "0", 16)
        priority = int(values.get("VLAN-PRIORITY", "4") or "4")
    except ValueError:
        warnings.append(f"{where} has an unreadable APPID, VLAN-ID or VLAN-PRIORITY")
        return None
    if not MAC.match(mac):
        warnings.append(f"{where} has no valid MAC-Address")
        return None
    if not (0 <= app_id <= 0xFFFF and 0 <= vlan_id <= 0xFFF and 0 <= priority <= 7):
        warnings.append(f"{where} has APPID, VLAN-ID or VLAN-PRIORITY out of range")
        return None
    return {"ld_inst": element.get("ldInst", ""), "cb_name": element.get("cbName", ""),
            "mac": mac.upper(), "app_id": app_id, "vlan_id": vlan_id, "vlan_priority": priority,
            "min_time": _milliseconds(element.find("MinTime")),
            "max_time": _milliseconds(element.find("MaxTime"))}


def read_communication(root, ied_name):
    """Return the IED's SCD network address and GSE addresses, plus warnings."""
    warnings = []
    result = {"address": None, "gse": []}
    for subnetwork in root.findall("Communication/SubNetwork"):
        for access in subnetwork.findall("ConnectedAP"):
            if access.get("iedName") != ied_name:
                continue
            values = _parameters(access.find("Address"))
            ip = _ip(values.get("IP"))
            if ip and result["address"] is None:
                result["address"] = {"ip": ip, "subnet": _ip(values.get("IP-SUBNET")),
                                     "gateway": _ip(values.get("IP-GATEWAY")),
                                     "access_point": access.get("apName", ""),
                                     "subnetwork": subnetwork.get("name", "")}
            elif values.get("IP") and not ip:
                warnings.append(f"SCD address for {ied_name} is not a valid IPv4 address")
            for element in access.findall("GSE"):
                parsed = _gse(element, warnings, ied_name)
                if parsed:
                    result["gse"].append(parsed)
    return result, warnings
