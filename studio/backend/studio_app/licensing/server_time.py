"""Verify the license server's signed time stamps (same Ed25519 keys and kid as certificates).

A stamp payload is {"kid", "purpose": "server-time", "time", "version": 1}. It has
no "type" or "hwid", so it can never pass as a license, and a license never passes as a
stamp. Native twin: license_clock.cpp.
"""
from . import certificate
from .certificate import CertificateError

PURPOSE = "server-time"


def verify(stamp, keys=None):
    """Return the time of a genuine stamp, or None for missing, foreign or altered input."""
    if not stamp:
        return None
    try:
        payload = certificate.verify_bundle(certificate.parse_bundle(stamp), keys)
        if payload.get("purpose") != PURPOSE or int(payload.get("version", 0)) != 1:
            return None
        moment = int(payload["time"])
    except (CertificateError, KeyError, TypeError, ValueError, AttributeError):
        return None
    return moment if moment > 0 else None
