"""Read Windows network adapters and IPv4 addresses through the IP Helper API.

One call takes milliseconds, unlike the PowerShell cmdlets, so the adapter list can be
refreshed live. Read-only: nothing here changes adapter settings.
"""
import ctypes
import os
import socket
from ctypes import wintypes

AF_INET = 2
GAA_FLAGS = 0x2 | 0x4 | 0x8 | 0x80  # skip anycast, multicast, DNS servers; include gateways
ERROR_BUFFER_OVERFLOW = 111
IF_TYPE_SOFTWARE_LOOPBACK, IF_TYPE_TUNNEL = 24, 131
HIDDEN_WORDS = ("wi-fi direct virtual adapter", "wan miniport", "kernel debug")
PREFIX_ORIGINS = {0: "Other", 1: "Manual", 2: "WellKnown", 3: "Dhcp", 4: "RouterAdvertisement"}


class _SocketAddress(ctypes.Structure):
    _fields_ = [("lpSockaddr", ctypes.c_void_p), ("iSockaddrLength", ctypes.c_int)]


class _Unicast(ctypes.Structure):
    pass


_Unicast._fields_ = [("Length", wintypes.ULONG), ("Flags", wintypes.DWORD),
                     ("Next", ctypes.POINTER(_Unicast)), ("Address", _SocketAddress),
                     ("PrefixOrigin", ctypes.c_int), ("SuffixOrigin", ctypes.c_int),
                     ("DadState", ctypes.c_int), ("ValidLifetime", wintypes.ULONG),
                     ("PreferredLifetime", wintypes.ULONG), ("LeaseLifetime", wintypes.ULONG),
                     ("OnLinkPrefixLength", ctypes.c_uint8)]


class _Adapter(ctypes.Structure):
    pass


# IP_ADAPTER_ADDRESSES_LH up to FirstGatewayAddress (the fields this module reads).
_Adapter._fields_ = [("Length", wintypes.ULONG), ("IfIndex", wintypes.DWORD),
                     ("Next", ctypes.POINTER(_Adapter)), ("AdapterName", ctypes.c_char_p),
                     ("FirstUnicastAddress", ctypes.POINTER(_Unicast)),
                     ("FirstAnycastAddress", ctypes.c_void_p), ("FirstMulticastAddress", ctypes.c_void_p),
                     ("FirstDnsServerAddress", ctypes.c_void_p), ("DnsSuffix", ctypes.c_wchar_p),
                     ("Description", ctypes.c_wchar_p), ("FriendlyName", ctypes.c_wchar_p),
                     ("PhysicalAddress", ctypes.c_ubyte * 8), ("PhysicalAddressLength", wintypes.ULONG),
                     ("Flags", wintypes.ULONG), ("Mtu", wintypes.ULONG), ("IfType", wintypes.DWORD),
                     ("OperStatus", ctypes.c_int), ("Ipv6IfIndex", wintypes.DWORD),
                     ("ZoneIndices", wintypes.ULONG * 16), ("FirstPrefix", ctypes.c_void_p),
                     ("TransmitLinkSpeed", ctypes.c_uint64), ("ReceiveLinkSpeed", ctypes.c_uint64),
                     ("FirstWinsServerAddress", ctypes.c_void_p), ("FirstGatewayAddress", ctypes.c_void_p)]


class _UnicastRow(ctypes.Structure):
    """MIB_UNICASTIPADDRESS_ROW (80 bytes on x64)."""
    _fields_ = [("Address", ctypes.c_ubyte * 28), ("InterfaceLuid", ctypes.c_uint64),
                ("InterfaceIndex", wintypes.ULONG), ("PrefixOrigin", ctypes.c_int),
                ("SuffixOrigin", ctypes.c_int), ("ValidLifetime", wintypes.ULONG),
                ("PreferredLifetime", wintypes.ULONG), ("OnLinkPrefixLength", ctypes.c_uint8),
                ("SkipAsSource", ctypes.c_uint8), ("DadState", ctypes.c_int),
                ("ScopeId", wintypes.ULONG), ("CreationTimeStamp", ctypes.c_int64)]


class _UnicastTable(ctypes.Structure):
    _fields_ = [("NumEntries", wintypes.ULONG), ("Table", _UnicastRow * 1)]


def status_word(oper_status):
    """Same words as Get-NetAdapter: Up, Disconnected (no cable/link) or Down."""
    return {1: "Up", 2: "Disconnected", 7: "Disconnected"}.get(oper_status, "Down")


def speed_words(bits_per_second):
    if not bits_per_second or bits_per_second >= 2 ** 63:
        return "0 bps"
    for unit, size in (("Gbps", 10 ** 9), ("Mbps", 10 ** 6), ("Kbps", 10 ** 3)):
        if bits_per_second >= size:
            return f"{bits_per_second / size:g} {unit}"
    return f"{bits_per_second} bps"


def mac_words(raw, length):
    return "-".join(f"{byte:02X}" for byte in bytes(raw)[:length])


def is_listed(if_type, description):
    """True for adapters Get-NetAdapter shows (no loopback, tunnels or hidden miniports)."""
    if if_type in (IF_TYPE_SOFTWARE_LOOPBACK, IF_TYPE_TUNNEL):
        return False
    text = (description or "").lower()
    return not any(word in text for word in HIDDEN_WORDS)


def _iphlpapi():
    return ctypes.WinDLL("iphlpapi")


def _adapter_buffer(library):
    size = wintypes.ULONG(64 * 1024)
    for _ in range(3):
        buffer = ctypes.create_string_buffer(size.value)
        result = library.GetAdaptersAddresses(AF_INET, GAA_FLAGS, None, buffer, ctypes.byref(size))
        if result == 0:
            return buffer
        if result != ERROR_BUFFER_OVERFLOW:
            raise OSError(result, "GetAdaptersAddresses failed")
    raise OSError(ERROR_BUFFER_OVERFLOW, "GetAdaptersAddresses buffer kept growing")


def _ipv4(socket_address):
    raw = ctypes.string_at(socket_address.lpSockaddr, 8)
    return socket.inet_ntoa(raw[4:8])


def unicast_rows(library=None):
    """{(ifIndex, ip): {prefix, origin, skip_as_source}} for every IPv4 address."""
    library = library or _iphlpapi()
    pointer = ctypes.POINTER(_UnicastTable)()
    result = library.GetUnicastIpAddressTable(AF_INET, ctypes.byref(pointer))
    if result != 0:
        raise OSError(result, "GetUnicastIpAddressTable failed")
    try:
        count = pointer.contents.NumEntries
        rows = ctypes.cast(ctypes.addressof(pointer.contents.Table), ctypes.POINTER(_UnicastRow * count)).contents
        found = {}
        for row in rows:
            ip = socket.inet_ntoa(bytes(row.Address[4:8]))
            found[(row.InterfaceIndex, ip)] = {"prefix": row.OnLinkPrefixLength,
                                               "origin": PREFIX_ORIGINS.get(row.PrefixOrigin, "Other"),
                                               "skip_as_source": bool(row.SkipAsSource)}
        return found
    finally:
        library.FreeMibTable(pointer)


def _walk(buffer):
    adapter = ctypes.cast(buffer, ctypes.POINTER(_Adapter))
    while adapter:
        yield adapter.contents
        adapter = adapter.contents.Next


def _unicasts(item):
    unicast = item.FirstUnicastAddress
    while unicast:
        yield unicast.contents
        unicast = unicast.contents.Next


def adapters():
    """Raw adapter dictionaries in the same shape the PowerShell listing produced."""
    if os.name != "nt":
        return []
    library = _iphlpapi()
    buffer = _adapter_buffer(library)
    details = unicast_rows(library)
    result = []
    for item in _walk(buffer):
        if not is_listed(item.IfType, item.Description):
            continue
        addresses = []
        for unicast in _unicasts(item):
            ip = _ipv4(unicast.Address)
            extra = details.get((item.IfIndex, ip)) or {
                "prefix": unicast.OnLinkPrefixLength, "skip_as_source": False,
                "origin": PREFIX_ORIGINS.get(unicast.PrefixOrigin, "Other")}
            addresses.append({"ip": ip, **extra})
        result.append({"guid": (item.AdapterName or b"").decode(), "name": item.FriendlyName or "",
                       "description": item.Description or "", "status": status_word(item.OperStatus),
                       "index": item.IfIndex, "if_type": item.IfType,
                       "mac": mac_words(item.PhysicalAddress, item.PhysicalAddressLength),
                       "speed": speed_words(item.TransmitLinkSpeed),
                       "default_route": bool(item.FirstGatewayAddress), "addresses": addresses})
    return result


def addresses_on(adapter_guid):
    """IPv4 addresses on the adapter with this GUID (empty when not found)."""
    if os.name != "nt":
        return set()
    wanted = str(adapter_guid).upper()
    for item in _walk(_adapter_buffer(_iphlpapi())):
        if (item.AdapterName or b"").decode().upper() == wanted:
            return {_ipv4(unicast.Address) for unicast in _unicasts(item)}
    return set()
