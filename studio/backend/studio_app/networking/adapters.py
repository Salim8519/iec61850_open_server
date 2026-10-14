"""List Windows network adapters and their IPv4 addresses without elevation."""
import json
import logging
import os
import subprocess
from . import ip_helper

LOG = logging.getLogger(__name__)

SCRIPT = r"""
$ErrorActionPreference = 'Stop'
$routes = @(Get-NetRoute -DestinationPrefix '0.0.0.0/0' -ErrorAction SilentlyContinue |
  Select-Object -ExpandProperty ifIndex)
$result = @(Get-NetAdapter | ForEach-Object {
  $a = $_
  $ips = @(Get-NetIPAddress -InterfaceIndex $a.ifIndex -AddressFamily IPv4 -ErrorAction SilentlyContinue |
    ForEach-Object { @{ ip = $_.IPAddress; prefix = [int]$_.PrefixLength;
      skip_as_source = [bool]$_.SkipAsSource; origin = [string]$_.PrefixOrigin } })
  @{ guid = [string]$a.InterfaceGuid; name = $a.Name; description = $a.InterfaceDescription;
     status = [string]$a.Status; index = [int]$a.ifIndex; mac = $a.MacAddress;
     speed = [string]$a.LinkSpeed; virtual = [bool]$a.Virtual;
     default_route = ($routes -contains $a.ifIndex); addresses = $ips }
})
ConvertTo-Json -InputObject $result -Depth 5 -Compress
"""


def _powershell(script, timeout=20):
    flags = subprocess.CREATE_NO_WINDOW if os.name == "nt" else 0
    completed = subprocess.run(["powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script],
                               capture_output=True, text=True, timeout=timeout, creationflags=flags)
    if completed.returncode != 0:
        raise RuntimeError("Windows could not list network adapters: "
                           + (completed.stderr.strip().splitlines() or ["unknown error"])[-1])
    return completed.stdout


def _kind(item):
    text = (item.get("description", "") + " " + item.get("name", "")).lower()
    if "loopback" in text and "km-test" in text:
        return "virtual_lab"
    if any(word in text for word in ("vmware", "virtualbox", "hyper-v", "vethernet", "km-test")):
        return "virtual"
    if any(word in text for word in ("wi-fi", "wireless", "wlan", "802.11")):
        return "wifi"
    if any(word in text for word in ("vpn", "zerotier", "tap-", "tunnel", "wireguard", "openvpn")):
        return "vpn"
    if "bluetooth" in text:
        return "bluetooth"
    return "ethernet"


def normalize(raw):
    """Turn adapter records (IP Helper list or PowerShell JSON) into stable dictionaries."""
    items = json.loads(raw or "[]") if isinstance(raw, (str, type(None))) else raw
    if isinstance(items, dict):
        items = [items]
    adapters = []
    for item in items:
        addresses = item.get("addresses") or []
        if isinstance(addresses, dict):
            addresses = [addresses]
        guid = str(item.get("guid", "")).upper()
        if not guid.startswith("{"):
            guid = "{" + guid + "}"
        adapters.append({"guid": guid, "name": item.get("name", ""),
                         "description": item.get("description", ""), "status": item.get("status", ""),
                         "index": item.get("index"), "mac": item.get("mac", ""), "speed": item.get("speed", ""),
                         "kind": _kind(item), "default_route": bool(item.get("default_route")),
                         "addresses": [{"ip": a.get("ip"), "prefix": a.get("prefix"),
                                        "skip_as_source": bool(a.get("skip_as_source")),
                                        "origin": a.get("origin", "")} for a in addresses]})
    return sorted(adapters, key=lambda a: (KIND_ORDER.get(a["kind"], 9), a["status"] != "Up", a["default_route"],
                                           a["name"].lower()))


KIND_ORDER = {"ethernet": 0, "wifi": 1, "virtual_lab": 2, "virtual": 3, "vpn": 4, "bluetooth": 5}


def list_adapters(run=_powershell, native=ip_helper.adapters):
    """Adapters from the Windows IP Helper API (milliseconds); PowerShell only if that fails."""
    try:
        return normalize(native())
    except OSError as error:
        LOG.warning("IP Helper adapter listing failed (%s); using PowerShell", error)
        return normalize(run(SCRIPT))


RULES_SCRIPT = ("Get-NetFirewallRule -Name 'SubstationStudio-MMS-*' -ErrorAction SilentlyContinue | "
                "Select-Object -ExpandProperty Name")


def list_studio_rules(run=_powershell):
    """Names of Windows Firewall rules created by Studio that still exist."""
    return [line.strip() for line in run(RULES_SCRIPT).splitlines() if line.strip()]
