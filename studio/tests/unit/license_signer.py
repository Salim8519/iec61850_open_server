"""Test-only Ed25519 signer (RFC 8032 section 6) for building license certificates."""
import base64
import hashlib
import json
from studio_app.licensing import ed25519_verify as ed
from studio_app.licensing.release_key import key_id

TEST_SEED = bytes.fromhex("42" * 32)
HWID = "ab" * 32


def _secret(seed):
    digest = hashlib.sha512(seed).digest()
    scalar = int.from_bytes(digest[:32], "little")
    scalar &= (1 << 254) - 8
    scalar |= 1 << 254
    return scalar, digest[32:]


def _compress(point):
    inverse = ed._inverse(point[2])
    x, y = point[0] * inverse % ed.P, point[1] * inverse % ed.P
    return int.to_bytes(y | ((x & 1) << 255), 32, "little")


def public_key(seed=TEST_SEED):
    return _compress(ed._multiply(_secret(seed)[0], ed.BASE))


def sign(message, seed=TEST_SEED):
    scalar, prefix = _secret(seed)
    public = _compress(ed._multiply(scalar, ed.BASE))
    r = int.from_bytes(hashlib.sha512(prefix + message).digest(), "little") % ed.Q
    encoded_r = _compress(ed._multiply(r, ed.BASE))
    h = int.from_bytes(hashlib.sha512(encoded_r + public + message).digest(), "little") % ed.Q
    return encoded_r + int.to_bytes((r + h * scalar) % ed.Q, 32, "little")


def keys(seed=TEST_SEED):
    public = public_key(seed)
    return {key_id(public): public}


def bundle(seed=TEST_SEED, **fields):
    """Return license.cert JSON text signed like the Go server signs it."""
    payload = {"email": "", "expires_at": 0, "hwid": HWID, "issued_at": 1_760_000_000,
               "kid": key_id(public_key(seed)), "license_id": "SUBS-TEST-TEST-TEST-TEST",
               "max_machines": 1, "type": "lifetime", "version": 1}
    payload.update(fields)
    text = json.dumps(payload, separators=(",", ":"), sort_keys=True)
    signature = base64.urlsafe_b64encode(sign(text.encode("utf-8"), seed)).decode().rstrip("=")
    return json.dumps({"payload": text, "signature": signature})


def stamp(moment, seed=TEST_SEED):
    """A license-server time stamp, signed like certs.SignTime (returns the bundle dict)."""
    payload = {"kid": key_id(public_key(seed)), "purpose": "server-time", "time": moment, "version": 1}
    text = json.dumps(payload, separators=(",", ":"), sort_keys=True)
    signature = base64.urlsafe_b64encode(sign(text.encode("utf-8"), seed)).decode().rstrip("=")
    return {"payload": text, "signature": signature}
