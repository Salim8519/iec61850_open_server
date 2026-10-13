"""Ask the local network whether another device already answers on an address."""
import ctypes
import ipaddress
import os
import socket
from concurrent.futures import ThreadPoolExecutor


def _answers(address):
    if os.name != "nt":
        return False
    iphlpapi = ctypes.WinDLL("iphlpapi")
    destination = int.from_bytes(socket.inet_aton(address), "little")
    mac = (ctypes.c_ubyte * 8)()
    length = ctypes.c_ulong(8)
    return iphlpapi.SendARP(destination, 0, ctypes.byref(mac), ctypes.byref(length)) == 0 and length.value > 0


def addresses_in_use(addresses, probe=_answers):
    """Return the subset of IPv4 addresses that another device already answers on."""
    candidates = [str(ipaddress.IPv4Address(a)) for a in addresses if a]
    if not candidates:
        return set()
    with ThreadPoolExecutor(max_workers=min(16, len(candidates))) as pool:
        answered = list(pool.map(probe, candidates))
    return {address for address, used in zip(candidates, answered) if used}
