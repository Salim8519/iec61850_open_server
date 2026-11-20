"""TCP sockets in LISTEN state with their owning process (IP Helper GetExtendedTcpTable).

A simulated IED's worker listens on its MMS address once its model has started, so the first
moment Windows lists that socket is the end of "model load" as Windows sees it.
Usage: `listeners(prefix="198.18.20.")` -> {(ip, port): pid}.
"""
import ctypes
from ctypes import wintypes
import socket
import struct

AF_INET = 2
TCP_TABLE_OWNER_PID_LISTENER = 3


class _Row(ctypes.Structure):  # MIB_TCPROW_OWNER_PID
    _fields_ = [("State", wintypes.DWORD), ("LocalAddr", wintypes.DWORD), ("LocalPort", wintypes.DWORD),
                ("RemoteAddr", wintypes.DWORD), ("RemotePort", wintypes.DWORD), ("OwningPid", wintypes.DWORD)]


_iphlp = ctypes.WinDLL("iphlpapi")
_iphlp.GetExtendedTcpTable.argtypes = [ctypes.c_void_p, ctypes.POINTER(wintypes.DWORD), wintypes.BOOL,
                                       wintypes.ULONG, ctypes.c_int, wintypes.ULONG]
_buffer = ctypes.create_string_buffer(1 << 16)


def listeners(prefix=""):
    """{(ip, port): owning pid} of every IPv4 TCP listener whose address starts with prefix."""
    global _buffer
    size = wintypes.DWORD(len(_buffer))
    while True:
        error = _iphlp.GetExtendedTcpTable(_buffer, ctypes.byref(size), False, AF_INET,
                                           TCP_TABLE_OWNER_PID_LISTENER, 0)
        if error == 122:  # ERROR_INSUFFICIENT_BUFFER
            _buffer = ctypes.create_string_buffer(size.value + 4096)
            size = wintypes.DWORD(len(_buffer))
            continue
        if error:
            raise OSError(error, "GetExtendedTcpTable failed")
        break
    count = struct.unpack_from("<I", _buffer, 0)[0]
    rows = (_Row * count).from_buffer_copy(_buffer, 4)[:count] if count else []
    found = {}
    for row in rows:
        ip = socket.inet_ntoa(struct.pack("<I", row.LocalAddr))
        if ip.startswith(prefix):
            found[(ip, socket.ntohs(row.LocalPort & 0xFFFF))] = row.OwningPid
    return found


if __name__ == "__main__":
    for (ip, port), pid in sorted(listeners().items()):
        print(f"{ip}:{port} pid {pid}")
