"""Serve the browser UI from the folder build or from the packed frontend.pak.

Protected builds ship one frontend.pak instead of loose .js/.html files. It is read
once into memory at startup. The masking key is in this compiled module, so this
deters casual copying of the UI; it is not a secret against a determined attacker.
Development serves frontend/dist directly; both paths return identical bytes.
"""
import hashlib
import json
import mimetypes
from pathlib import Path
import struct
import zlib
from flask import Response, send_from_directory

MAGIC = b"SSPAK1"
_STREAM_KEY = bytes.fromhex("8c1e5a7f03d94b6e2fa1c0d7e95b3418a6f20c7d51e9b38f4a0d6c2e97b15f83")


def keystream_xor(data, nonce):
    """XOR data with a SHAKE-256(key | nonce) keystream (symmetric, a few ms per MB)."""
    if not data:
        return b""
    pad = hashlib.shake_256(_STREAM_KEY + nonce).digest(len(data))
    return (int.from_bytes(data, "little") ^ int.from_bytes(pad, "little")).to_bytes(len(data), "little")


def pack(files):
    """files: {relative_path: bytes} -> frontend.pak bytes."""
    index, blobs, offset = {}, [], 0
    for name in sorted(files):
        compressed = zlib.compress(files[name], 9)
        index[name] = [offset, len(compressed)]
        blobs.append(compressed)
        offset += len(compressed)
    header = json.dumps(index, separators=(",", ":")).encode("utf-8")
    body = struct.pack("<I", len(header)) + header + b"".join(blobs)
    nonce = hashlib.sha256(body).digest()[:16]
    return MAGIC + nonce + keystream_xor(body, nonce)


def unpack(blob):
    """frontend.pak bytes -> {relative_path: bytes}."""
    if not blob.startswith(MAGIC):
        raise ValueError("frontend.pak is not a ScadaBench frontend package")
    nonce = blob[len(MAGIC):len(MAGIC) + 16]
    body = keystream_xor(blob[len(MAGIC) + 16:], nonce)
    (size,) = struct.unpack("<I", body[:4])
    index = json.loads(body[4:4 + size])
    data = body[4 + size:]
    return {name: zlib.decompress(data[start:start + length]) for name, (start, length) in index.items()}


def _mime(name):
    guessed = mimetypes.guess_type(name)[0]
    if name.endswith(".js"):
        return "text/javascript"
    return guessed or "application/octet-stream"


class FrontendAssets:
    def __init__(self, location):
        self.location = Path(location)
        self.files = None
        package = self.location if self.location.suffix == ".pak" else self.location / "frontend.pak"
        if package.is_file():
            self.files = unpack(package.read_bytes())

    def available(self):
        if self.files is not None:
            return "index.html" in self.files
        return (self.location / "index.html").exists()

    def _memory(self, name):
        content = self.files.get(name)
        if content is None:
            return Response("Not found", status=404)
        response = Response(content, mimetype=_mime(name))
        response.headers["Cache-Control"] = "no-cache" if name == "index.html" else "public, max-age=31536000, immutable"
        return response

    def index(self):
        if self.files is not None:
            return self._memory("index.html")
        return send_from_directory(self.location, "index.html")

    def asset(self, filename):
        if self.files is not None:
            return self._memory("assets/" + filename)
        return send_from_directory(self.location / "assets", filename)
