"""Ed25519 signing (RFC 8032), dependency-free, reusing ed25519_verify's field math.

Used by the build to sign the compiled backend module so the native workers can
verify it (integrity.cpp verify_module). Kept out of the shipped app: only the build
and tests sign; the workers and runtime only ever verify.
"""
import hashlib
from studio_app.licensing import ed25519_verify as ed


def _clamp(seed):
    digest = hashlib.sha512(seed).digest()
    scalar = int.from_bytes(digest[:32], "little")
    scalar &= (1 << 254) - 8
    scalar |= 1 << 254
    return scalar, digest[32:]


def _compress(point):
    inverse = ed._inverse(point[2])
    x, y = point[0] * inverse % ed.P, point[1] * inverse % ed.P
    return int.to_bytes(y | ((x & 1) << 255), 32, "little")


def public_key(seed):
    """Return the 32-byte compressed Ed25519 public key for a 32-byte seed."""
    return _compress(ed._multiply(_clamp(seed)[0], ed.BASE))


def sign(message, seed):
    """Return the 64-byte Ed25519 signature of message bytes under a 32-byte seed."""
    scalar, prefix = _clamp(seed)
    public = _compress(ed._multiply(scalar, ed.BASE))
    r = int.from_bytes(hashlib.sha512(prefix + message).digest(), "little") % ed.Q
    encoded_r = _compress(ed._multiply(r, ed.BASE))
    h = int.from_bytes(hashlib.sha512(encoded_r + public + message).digest(), "little") % ed.Q
    return encoded_r + int.to_bytes((r + h * scalar) % ed.Q, 32, "little")
