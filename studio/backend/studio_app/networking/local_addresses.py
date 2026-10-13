"""Fast lookup of the IPv4 addresses currently present on one Windows adapter."""
import ctypes
import os
import socket
from ctypes import wintypes

AF_INET = 2
GAA_FLAG_SKIP_ANYCAST, GAA_FLAG_SKIP_MULTICAST, GAA_FLAG_SKIP_DNS_SERVER = 0x2, 0x4, 0x8


class _SocketAddress(ctypes.Structure):
    _fields_ = [("lpSockaddr", ctypes.c_void_p), ("iSockaddrLength", ctypes.c_int)]


class _Unicast(ctypes.Structure):
    pass


_Unicast._fields_ = [("Length", wintypes.ULONG), ("Flags", wintypes.DWORD),
                     ("Next", ctypes.POINTER(_Unicast)), ("Address", _SocketAddress)]


class _Adapter(ctypes.Structure):
    pass


_Adapter._fields_ = [("Length", wintypes.ULONG), ("IfIndex", wintypes.DWORD),
                     ("Next", ctypes.POINTER(_Adapter)), ("AdapterName", ctypes.c_char_p),
                     ("FirstUnicastAddress", ctypes.POINTER(_Unicast))]


def addresses_on(adapter_guid):
    """IPv4 addresses on the adapter with this GUID (empty when not found)."""
    if os.name != "nt":
        return set()
    wanted = str(adapter_guid).upper()
    size = wintypes.ULONG(64 * 1024)
    buffer = ctypes.create_string_buffer(size.value)
    flags = GAA_FLAG_SKIP_ANYCAST | GAA_FLAG_SKIP_MULTICAST | GAA_FLAG_SKIP_DNS_SERVER
    call = ctypes.windll.iphlpapi.GetAdaptersAddresses
    if call(AF_INET, flags, None, buffer, ctypes.byref(size)) == 111:
        buffer = ctypes.create_string_buffer(size.value)
        if call(AF_INET, flags, None, buffer, ctypes.byref(size)) != 0:
            return set()
    result = set()
    adapter = ctypes.cast(buffer, ctypes.POINTER(_Adapter))
    while adapter:
        if (adapter.contents.AdapterName or b"").decode().upper() == wanted:
            unicast = adapter.contents.FirstUnicastAddress
            while unicast:
                raw = ctypes.string_at(unicast.contents.Address.lpSockaddr, 8)
                result.add(socket.inet_ntoa(raw[4:8]))
                unicast = unicast.contents.Next
        adapter = adapter.contents.Next
    return result
