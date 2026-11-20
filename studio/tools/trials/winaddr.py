"""The real Windows address list (IP Helper GetUnicastIpAddressTable), never the project or journal.

Each row carries Windows' own creation time of the address, so a trial can tell when Windows
itself had an address, independent of when the engine says it asked for it.
Usage: `addresses(prefix="198.18.20.")` -> {ip: Row}; `lab_card_index()` -> interface index or None.
"""
import ctypes
from ctypes import wintypes
from dataclasses import dataclass
import socket

from winclock import filetime_to_unix

AF_INET = 2
LAB_CARD_ALIAS = "ScadaBench Lab"
DAD_STATES = {0: "invalid", 1: "tentative", 2: "duplicate", 3: "deprecated", 4: "preferred"}


class _Row(ctypes.Structure):  # MIB_UNICASTIPADDRESS_ROW, 80 bytes on x64
    _fields_ = [("Address", ctypes.c_ubyte * 28), ("InterfaceLuid", ctypes.c_uint64),
                ("InterfaceIndex", wintypes.ULONG), ("PrefixOrigin", ctypes.c_int),
                ("SuffixOrigin", ctypes.c_int), ("ValidLifetime", wintypes.ULONG),
                ("PreferredLifetime", wintypes.ULONG), ("OnLinkPrefixLength", ctypes.c_ubyte),
                ("SkipAsSource", ctypes.c_ubyte), ("DadState", ctypes.c_int), ("ScopeId", wintypes.ULONG),
                ("CreationTimeStamp", ctypes.c_int64)]


assert ctypes.sizeof(_Row) == 80, "MIB_UNICASTIPADDRESS_ROW layout changed"

_iphlp = ctypes.WinDLL("iphlpapi")
_iphlp.GetUnicastIpAddressTable.argtypes = [ctypes.c_ushort, ctypes.POINTER(ctypes.c_void_p)]
_iphlp.FreeMibTable.argtypes = [ctypes.c_void_p]
_iphlp.ConvertInterfaceLuidToAlias.argtypes = [ctypes.POINTER(ctypes.c_uint64), ctypes.c_wchar_p, ctypes.c_size_t]


@dataclass(frozen=True)
class Row:
    ip: str
    interface: int
    state: str          # DAD state: tentative / preferred / duplicate ...
    created: float      # Windows' creation time, Unix seconds
    skip_as_source: bool


def _alias(luid):
    buffer = ctypes.create_unicode_buffer(257)
    value = ctypes.c_uint64(luid)
    if _iphlp.ConvertInterfaceLuidToAlias(ctypes.byref(value), buffer, len(buffer)):
        return ""
    return buffer.value


def _rows():
    table = ctypes.c_void_p()
    error = _iphlp.GetUnicastIpAddressTable(AF_INET, ctypes.byref(table))
    if error:
        raise OSError(error, "GetUnicastIpAddressTable failed")
    try:
        count = ctypes.cast(table, ctypes.POINTER(wintypes.ULONG))[0]
        rows = ctypes.cast(table.value + 8, ctypes.POINTER(_Row * count)).contents  # NumEntries + padding
        return [(row, bytes(row.Address[4:8])) for row in rows]
    finally:
        _iphlp.FreeMibTable(table)


def addresses(prefix="", interface=None):
    """{ip: Row} of every IPv4 address Windows holds now, optionally filtered."""
    found = {}
    for row, raw in _rows():
        ip = socket.inet_ntoa(raw)
        if ip.startswith(prefix) and (interface is None or row.InterfaceIndex == interface):
            found[ip] = Row(ip, row.InterfaceIndex, DAD_STATES.get(row.DadState, str(row.DadState)),
                            filetime_to_unix(row.CreationTimeStamp), bool(row.SkipAsSource))
    return found


def card_index(alias):
    """Interface index of the card Windows names `alias` (e.g. "Ethernet"), or None."""
    for row, _ in _rows():
        if _alias(row.InterfaceLuid) == alias:
            return row.InterfaceIndex
    return None


def lab_card_index():
    """Interface index of the ScadaBench Lab card, or None when Windows has no such card."""
    return card_index(LAB_CARD_ALIAS)


if __name__ == "__main__":
    card = lab_card_index()
    print(f"Lab card interface index: {card}")
    for item in addresses(interface=card).values():
        print(item)
