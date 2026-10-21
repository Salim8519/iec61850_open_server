"""An independent "real device" on a lab adapter: a raw Npcap handle driven from Python (ADR-0073).

It shares no code with the GOOSE engine, so what it sees on the adapter is outside evidence.
"""
import ctypes
import os
import time
from pathlib import Path

NPCAP = Path(os.environ.get("SystemRoot", r"C:\Windows")) / "System32" / "Npcap"
ADAPTER = os.environ.get("STUDIO_GOOSE_ADAPTER", "")
AVAILABLE = (NPCAP / "wpcap.dll").exists()
MULTICAST = bytes.fromhex("010ccd010001")


class Header(ctypes.Structure):
    _fields_ = [("seconds", ctypes.c_long), ("micros", ctypes.c_long),
                ("caplen", ctypes.c_uint), ("length", ctypes.c_uint)]


class Program(ctypes.Structure):
    _fields_ = [("length", ctypes.c_uint), ("instructions", ctypes.c_void_p)]


def _library():
    # Load from the Npcap folder: an old WinPcap copy in System32 must never be picked up.
    os.add_dll_directory(str(NPCAP))
    ctypes.CDLL(str(NPCAP / "Packet.dll"))
    pcap = ctypes.CDLL(str(NPCAP / "wpcap.dll"))
    pcap.pcap_open_live.restype = ctypes.c_void_p
    pcap.pcap_open_live.argtypes = [ctypes.c_char_p, ctypes.c_int, ctypes.c_int, ctypes.c_int, ctypes.c_char_p]
    pcap.pcap_sendpacket.argtypes = [ctypes.c_void_p, ctypes.c_char_p, ctypes.c_int]
    pcap.pcap_next_ex.argtypes = [ctypes.c_void_p, ctypes.POINTER(ctypes.POINTER(Header)),
                                  ctypes.POINTER(ctypes.POINTER(ctypes.c_ubyte))]
    pcap.pcap_compile.argtypes = [ctypes.c_void_p, ctypes.POINTER(Program), ctypes.c_char_p, ctypes.c_int,
                                  ctypes.c_uint]
    pcap.pcap_setfilter.argtypes = [ctypes.c_void_p, ctypes.POINTER(Program)]
    pcap.pcap_geterr.restype = ctypes.c_char_p
    pcap.pcap_geterr.argtypes = [ctypes.c_void_p]
    pcap.pcap_close.argtypes = [ctypes.c_void_p]
    return pcap


class Peer:
    """Promiscuous GOOSE capture and injection on one adapter, by GUID."""

    def __init__(self, guid=ADAPTER):
        self.pcap = _library()
        error = ctypes.create_string_buffer(256)
        self.handle = self.pcap.pcap_open_live(f"\\Device\\NPF_{guid}".encode(), 65535, 1, 1, error)
        if not self.handle:
            raise OSError(error.value.decode(errors="replace"))
        program = Program()
        if self.pcap.pcap_compile(self.handle, ctypes.byref(program), b"ether proto 0x88b8 or vlan", 1,
                                  0xFFFFFFFF) == 0:
            self.pcap.pcap_setfilter(self.handle, ctypes.byref(program))

    def send(self, frame):
        if self.pcap.pcap_sendpacket(self.handle, frame, len(frame)) != 0:
            raise OSError(self.pcap.pcap_geterr(self.handle).decode(errors="replace"))

    def next(self):
        header, data = ctypes.POINTER(Header)(), ctypes.POINTER(ctypes.c_ubyte)()
        if self.pcap.pcap_next_ex(self.handle, ctypes.byref(header), ctypes.byref(data)) == 1:
            return ctypes.string_at(data, header.contents.caplen)
        return None

    def capture(self, seconds):
        """Every GOOSE frame heard for `seconds`."""
        frames, deadline = [], time.monotonic() + seconds
        while time.monotonic() < deadline:
            frame = self.next()
            if frame:
                frames.append(frame)
        return frames

    def close(self):
        self.pcap.pcap_close(self.handle)


def goose(source, gocb_ref, app_id=0x1000):
    """A minimal GOOSE frame: the engine reads only the addresses, APPID and gocbRef."""
    ref = gocb_ref.encode()
    inner = b"\x80" + bytes([len(ref)]) + ref
    pdu = b"\x61" + bytes([len(inner)]) + inner
    header = app_id.to_bytes(2, "big") + (8 + len(pdu)).to_bytes(2, "big") + b"\0\0\0\0"
    return MULTICAST + source + b"\x88\xb8" + header + pdu


def mac_bytes(text):
    return bytes.fromhex(text.replace("-", "").replace(":", ""))
