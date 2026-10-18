"""Validate project network mode and per-IED role, endpoint and GOOSE settings."""
import ipaddress
import re
from .test_mode import validate_test_mode

MODES = {"loopback", "lab"}
ROLES = {"simulated", "real", "off"}
ACCESS = {"auto_reports", "polling"}
GUID = re.compile(r"^\{[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}\}$")
DEFAULT_NETWORK = {"mode": "loopback", "adapter_guid": "", "adapter_label": "", "nominal_hz": 50}
NOMINAL_HZ = {50, 60}
DEFAULT_LIVE = {"ip": "", "port": 102, "access": "auto_reports", "poll_ms": 1000}


def _ipv4(value, label, allow_empty=True, allow_loopback=False):
    if value in ("", None) and allow_empty:
        return ""
    try:
        address = ipaddress.IPv4Address(str(value))
    except ValueError as error:
        raise ValueError(f"{label} must be an IPv4 address such as 192.168.1.20") from error
    if (address.is_loopback and not allow_loopback) or address.is_multicast or address.is_unspecified:
        raise ValueError(f"{label} must be a normal device address, not {address}")
    return str(address)


def _port(value, label):
    if isinstance(value, bool) or not isinstance(value, int) or not 1 <= value <= 65535:
        raise ValueError(f"{label} must be a whole number from 1 to 65535 (MMS normally uses 102)")
    return value


def validate_network(value):
    value = {**DEFAULT_NETWORK, **(value or {})}
    if value["mode"] not in MODES:
        raise ValueError("Network mode must be 'loopback' (this PC only) or 'lab' (network adapter)")
    guid = str(value.get("adapter_guid") or "").upper()
    if value["mode"] == "lab" and not GUID.match(guid):
        raise ValueError("Choose a network adapter before switching to lab network mode")
    if guid and not GUID.match(guid):
        raise ValueError("Adapter identifier is not a Windows adapter GUID")
    if value["nominal_hz"] not in NOMINAL_HZ or isinstance(value["nominal_hz"], bool):
        raise ValueError("Nominal frequency must be 50 Hz or 60 Hz")
    return {"mode": value["mode"], "adapter_guid": guid,
            "adapter_label": str(value.get("adapter_label") or "")[:100], "nominal_hz": value["nominal_hz"]}


def validate_live(value):
    value = {**DEFAULT_LIVE, **(value or {})}
    if value["access"] not in ACCESS:
        raise ValueError("Access must be 'auto_reports' (use a free report block) or 'polling'")
    poll = value["poll_ms"]
    if isinstance(poll, bool) or not isinstance(poll, int) or not 250 <= poll <= 60000:
        raise ValueError("Polling interval must be between 250 ms and 60 s")
    # A real device may also be another simulator on this PC, so loopback is allowed here.
    return {"ip": _ipv4(value["ip"], "Real IED address", allow_loopback=True),
            "port": _port(value["port"], "Real IED port"),
            "access": value["access"], "poll_ms": poll}


def validate_ied_settings(value):
    """Normalize the stored settings of one IED, filling format-11 defaults."""
    role = value.get("role", "simulated")
    if role not in ROLES:
        raise ValueError("IED role must be simulated, real or off")
    publish = value.get("goose_publish", True)
    if not isinstance(publish, bool):
        raise ValueError("GOOSE publishing must be on or off")
    return {"role": role, "live": validate_live(value.get("live")),
            "lab_ip": _ipv4(value.get("lab_ip", ""), "Lab network address"),
            "goose_publish": publish, "test_mode": validate_test_mode(value.get("test_mode"))}


def effective_endpoint(ied, network):
    """The address a simulated IED listens on in the current network mode."""
    if network["mode"] == "lab" and ied.get("lab_ip"):
        return ied["lab_ip"], ied.get("port", 102)
    return ied["ip"], ied.get("port", 102)


def real_endpoint(ied):
    """The address ScadaBench connects to for a real IED: user override, else SCD."""
    live = ied.get("live") or DEFAULT_LIVE
    scd = (ied.get("scl_address") or {}).get("ip", "")
    return (live["ip"] or scd), live["port"], ("override" if live["ip"] else "scd" if scd else "missing")


def command_refusal(ied):
    """Plain reason why ScadaBench will not command this IED, or None when it may."""
    role = ied.get("role", "simulated")
    if role == "real":
        return (f"{ied['name']} is a real device, so nothing was sent from here. Real equipment is "
                "operated only from Tools (IED Explorer or the gateway), after a confirmation.")
    if role == "off":
        return f"{ied['name']} is switched off in this project. Set it to 'Simulate' to operate it."
    return None
