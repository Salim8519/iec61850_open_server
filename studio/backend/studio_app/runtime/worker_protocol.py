"""Versioned binary framing for private native worker pipes."""
import json
import struct

MAX_FRAME = 16 * 1024 * 1024


def read_exact(stream, length):
    chunks = bytearray()
    while len(chunks) < length:
        data = stream.read(length - len(chunks))
        if not data:
            raise EOFError("Native worker pipe closed")
        chunks.extend(data)
    return bytes(chunks)


def read_frame(stream):
    length, = struct.unpack("<I", read_exact(stream, 4))
    if not 0 < length <= MAX_FRAME:
        raise ValueError(f"Invalid native frame length {length}")
    return json.loads(read_exact(stream, length))


def write_frame(stream, message):
    payload = json.dumps(message, allow_nan=False, separators=(",", ":")).encode("utf-8")
    if len(payload) > MAX_FRAME:
        raise ValueError("Native request exceeds frame limit")
    stream.write(struct.pack("<I", len(payload)) + payload)
    stream.flush()
