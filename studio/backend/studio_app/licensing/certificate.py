"""Parse a license.cert bundle and verify its signature against the release keys."""
import base64
import json
from . import ed25519_verify, release_key


class CertificateError(ValueError):
    """The bundle is unreadable, unsigned by a release key, or altered."""


def _decode_signature(text):
    padded = text.strip() + "=" * (-len(text.strip()) % 4)
    try:
        return base64.urlsafe_b64decode(padded)
    except (ValueError, TypeError) as error:
        raise CertificateError("The license file signature is unreadable.") from error


def parse_bundle(text):
    """Accept the license.cert JSON text (as saved or as pasted by a user)."""
    try:
        bundle = json.loads(text) if isinstance(text, str) else text
    except json.JSONDecodeError as error:
        raise CertificateError("This is not a license file. Paste the whole license.cert text.") from error
    if isinstance(bundle, dict) and isinstance(bundle.get("certificate"), dict):
        bundle = bundle["certificate"]
    if not isinstance(bundle, dict) or not isinstance(bundle.get("payload"), str) \
            or not isinstance(bundle.get("signature"), str):
        raise CertificateError("The license file is incomplete. Paste the whole license.cert text.")
    return {"payload": bundle["payload"], "signature": bundle["signature"]}


def verify_bundle(bundle, keys=None):
    """Return the payload dict when the signature matches an accepted key."""
    keys = release_key.public_keys() if keys is None else keys
    if not keys:
        raise CertificateError("This ScadaBench build has no license key configured. Contact the seller.")
    payload_bytes = bundle["payload"].encode("utf-8")
    try:
        payload = json.loads(payload_bytes)
    except json.JSONDecodeError as error:
        raise CertificateError("The license file is damaged.") from error
    if not isinstance(payload, dict):
        raise CertificateError("The license file is damaged.")
    signature = _decode_signature(bundle["signature"])
    candidates = [keys[payload["kid"]]] if payload.get("kid") in keys else list(keys.values())
    if not any(ed25519_verify.verify(key, payload_bytes, signature) for key in candidates):
        raise CertificateError("The license file is damaged or was changed. Activate again.")
    return payload
