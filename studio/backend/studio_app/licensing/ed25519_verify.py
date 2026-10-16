"""Ed25519 signature verification (RFC 8032 section 6 reference algorithm).

Verification only; it handles public data, so constant time is not required.
Kept dependency-free so the frozen build needs no extra native wheel. Correctness
is pinned by RFC 8032 test vectors and a certificate signed by Go's crypto/ed25519.
"""
import hashlib

P = 2**255 - 19
Q = 2**252 + 27742317777372353535851937790883648493
D = -121665 * pow(121666, P - 2, P) % P
SQRT_M1 = pow(2, (P - 1) // 4, P)


def _inverse(value):
    return pow(value, P - 2, P)


def _add(first, second):
    a = (first[1] - first[0]) * (second[1] - second[0]) % P
    b = (first[1] + first[0]) * (second[1] + second[0]) % P
    c = 2 * first[3] * second[3] * D % P
    d = 2 * first[2] * second[2] % P
    e, f, g, h = b - a, d - c, d + c, b + a
    return e * f % P, g * h % P, f * g % P, e * h % P


def _multiply(scalar, point):
    result = (0, 1, 1, 0)
    while scalar > 0:
        if scalar & 1:
            result = _add(result, point)
        point = _add(point, point)
        scalar >>= 1
    return result


def _equal(first, second):
    return ((first[0] * second[2] - second[0] * first[2]) % P == 0
            and (first[1] * second[2] - second[1] * first[2]) % P == 0)


def _recover_x(y, sign):
    if y >= P:
        return None
    x2 = (y * y - 1) * _inverse(D * y * y + 1) % P
    if x2 == 0:
        return None if sign else 0
    x = pow(x2, (P + 3) // 8, P)
    if (x * x - x2) % P != 0:
        x = x * SQRT_M1 % P
    if (x * x - x2) % P != 0:
        return None
    if (x & 1) != sign:
        x = P - x
    return x


def _decompress(encoded):
    if len(encoded) != 32:
        return None
    y = int.from_bytes(encoded, "little")
    sign = y >> 255
    y &= (1 << 255) - 1
    x = _recover_x(y, sign)
    if x is None:
        return None
    return x, y, 1, x * y % P


_GY = 4 * _inverse(5) % P
_GX = _recover_x(_GY, 0)
BASE = (_GX, _GY, 1, _GX * _GY % P)


def verify(public_key: bytes, message: bytes, signature: bytes) -> bool:
    """Return True only for a valid Ed25519 signature of message under public_key."""
    if len(public_key) != 32 or len(signature) != 64:
        return False
    point_a = _decompress(public_key)
    point_r = _decompress(signature[:32])
    if point_a is None or point_r is None:
        return False
    s = int.from_bytes(signature[32:], "little")
    if s >= Q:
        return False
    digest = hashlib.sha512(signature[:32] + public_key + message).digest()
    h = int.from_bytes(digest, "little") % Q
    return _equal(_multiply(s, BASE), _add(point_r, _multiply(h, point_a)))
