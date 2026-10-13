"""Refuse any helper request that is not exactly an allowed Studio network change."""
import hashlib
import ipaddress
import json
import re
from pathlib import Path
from studio_app.networking.change_plan import GROUP, RULE_PREFIX

GUID = re.compile(r"^\{[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}\}$")
RULE_NAME = re.compile(rf"^{re.escape(RULE_PREFIX)}[0-9a-f]{{12}}$")
ACTIONS = {"apply", "remove"}


class HelperRefused(Exception):
    """The request was refused before any Windows change was made."""


def _address(value):
    try:
        address = ipaddress.IPv4Address(value)
    except ValueError as error:
        raise HelperRefused(f"Not an IPv4 address: {value!r}") from error
    if address.is_loopback or address.is_multicast or address.is_link_local or address.is_unspecified:
        raise HelperRefused(f"Address {address} cannot be used for the lab network")
    return str(address)


def _alias(item):
    if not GUID.match(str(item.get("adapter_guid", ""))):
        raise HelperRefused("Alias item has no valid adapter GUID")
    index, prefix = item.get("adapter_index"), item.get("prefix")
    if isinstance(index, bool) or not isinstance(index, int) or index <= 0:
        raise HelperRefused("Alias item has no valid adapter index")
    if isinstance(prefix, bool) or not isinstance(prefix, int) or not 8 <= prefix <= 30:
        raise HelperRefused("Alias prefix must be between 8 and 30")
    _address(item.get("ip"))


def _firewall(item, program):
    if not RULE_NAME.match(str(item.get("name", ""))) or item.get("group") != GROUP:
        raise HelperRefused("Firewall item is not a Substation Studio rule")
    ports = item.get("ports")
    if not isinstance(ports, list) or not ports or any(
            isinstance(p, bool) or not isinstance(p, int) or not 1 <= p <= 65535 for p in ports):
        raise HelperRefused("Firewall ports are invalid")
    if not isinstance(item.get("addresses"), list) or not item["addresses"]:
        raise HelperRefused("Firewall rule needs at least one address")
    for value in item["addresses"]:
        _address(value)
    if Path(str(item.get("program", ""))).resolve() != Path(program).resolve():
        raise HelperRefused("Firewall rule must name the Substation Studio IED worker")


def validate_request(payload, expected_hash, program):
    """Return the parsed request if it is untampered and fully allowed."""
    if hashlib.sha256(payload).hexdigest() != expected_hash:
        raise HelperRefused("Request file changed after Studio prepared it")
    try:
        request = json.loads(payload)
    except ValueError as error:
        raise HelperRefused("Request file is not valid JSON") from error
    if request.get("action") not in ACTIONS or not isinstance(request.get("items"), list):
        raise HelperRefused("Unknown helper action")
    if not request["items"] or len(request["items"]) > 64:
        raise HelperRefused("Request must contain between 1 and 64 changes")
    for item in request["items"]:
        if not isinstance(item, dict) or not item.get("owned"):
            raise HelperRefused("Only changes owned by Studio can be applied or removed")
        if item.get("kind") == "alias":
            _alias(item)
        elif item.get("kind") == "firewall":
            _firewall(item, program)
        else:
            raise HelperRefused(f"Unknown change kind {item.get('kind')!r}")
    return request
