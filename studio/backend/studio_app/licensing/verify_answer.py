"""Check the license server's signed answer to POST /api/v1/verify.

The answer carries {"answer": {"payload": "<exact JSON text>", "signature": "<base64url>"}}
signed with the same Ed25519 key as certificates. The payload is
{"hwid", "kid", "license_id", "nonce", "purpose": "verify-answer", "status", "time",
"version": 1}. Only a genuine answer for THIS request counts: the signature must verify,
the purpose and version must match, and the nonce this client just sent, the license ID
and the machine ID must all come back unchanged. The unsigned top-level "status" is
never used, so a proxy or a hosts-file redirect cannot cancel (or keep alive) a license.
Anything else is treated like an unreachable server. Native twin: none (the native guard
never asks the server).
"""
from . import certificate
from .certificate import CertificateError

PURPOSE = "verify-answer"
STATUSES = ("valid", "revoked", "moved", "unknown")


def verified_status(answer, nonce, license_id, hwid, keys=None):
    """The signed verdict ("valid", "revoked", "moved" or "unknown"), or None if not genuine."""
    signed = answer.get("answer") if isinstance(answer, dict) else None
    if not isinstance(signed, dict) or not isinstance(signed.get("payload"), str) \
            or not isinstance(signed.get("signature"), str):
        return None
    try:
        payload = certificate.verify_bundle({"payload": signed["payload"], "signature": signed["signature"]}, keys)
    except (CertificateError, TypeError, KeyError):
        return None
    version = payload.get("version")
    if payload.get("purpose") != PURPOSE or type(version) is not int or version != 1:
        return None
    if not isinstance(license_id, str) or payload.get("license_id") != license_id:
        return None
    if payload.get("nonce") != nonce or payload.get("hwid") != hwid:
        return None
    status = payload.get("status")
    return status if isinstance(status, str) and status in STATUSES else None
