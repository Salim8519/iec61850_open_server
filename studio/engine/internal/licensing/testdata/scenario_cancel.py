"""Scenarios of ADR-0097 for make_scenarios.py: cancelled stays cancelled, only signed answers count.

A cancelled license ID lives in three places (a dot-file, HKCU\\Software\\ScadaBench and
license_state.json); putting the certificate back or deleting copies must never license the
PC again. A verify answer counts only when it is signed for the request just sent.
"""
from scenario_defs import CERT, DAY, HWID, LIFETIME, LIFETIME_ID, NONCE, NOW, STATE, TRIAL, TRIAL_ID, at, fail, ok, put, verdict
from scenario_trials import stamp, trial_verdict
from vector_common import OTHER_SEED, license_signer, signed

REVOKED_FILE = "home/.revoked_ids"
OTHER_ID = "SUBS-OTHR-OTHR-OTHR-OTHR"
LATER = NOW + 8 * DAY


def forged(**payload):
    """A revoked verdict whose signed answer is altered in one way (an unsigned "revoked" outside)."""
    return ok({"status": "revoked", "answer": license_signer.verify_answer(
        payload.pop("nonce", NONCE), payload.pop("license_id", LIFETIME_ID), payload.pop("machine", HWID),
        payload.pop("status", "revoked"), seed=payload.pop("seed", license_signer.TEST_SEED), **payload)})


BAD_ANSWERS = [
    ok({"status": "revoked"}), ok({"status": "revoked", "answer": None}), ok({"status": "revoked", "answer": "x"}),
    ok({"status": "revoked", "answer": {"payload": "{}"}}), ok({"status": "revoked", "answer": {"payload": 5, "signature": "s"}}),
    forged(nonce="f" * 32), forged(license_id=OTHER_ID), forged(machine="cd" * 32),
    forged(seed=OTHER_SEED), forged(purpose="server-time"), forged(version=2), forged(version=1.0),
    forged(version="1"), forged(status="banned"), forged(status=["revoked"]), ok([1]), ok("revoked"),
    fail("down", "offline"),
]

CANCELLED_SCENARIOS = [
    {"name": "cancelled stays cancelled", "registry": True, "client": {"verify": [ok(verdict("revoked"))]}, "steps": [
        put(CERT, LIFETIME), at(LATER, "verify"), put(CERT, LIFETIME), at(LATER, "status"), at(LATER, "licensed"),
        {"op": "remove", "path": REVOKED_FILE}, at(LATER + 1, "status"), put(STATE, "{}"), at(LATER + 2, "status"),
        {"op": "remove", "path": REVOKED_FILE}, put(STATE, '{"revoked_reason": null}'), at(LATER + 3, "licensed"),
        at(LATER + 4, "offline", arg=LIFETIME), at(LATER + 5, "status")]},
    {"name": "cancelled stays cancelled without a registry", "client": {"verify": [ok(verdict("moved"))]}, "steps": [
        put(CERT, LIFETIME), at(LATER, "verify"), put(CERT, LIFETIME), {"op": "remove", "path": REVOKED_FILE},
        at(LATER + 1, "status"), at(LATER + 2, "licensed")]},
    {"name": "other licenses are not cancelled", "registry": True, "client": {"verify": [ok(verdict("revoked"))]},
     "steps": [put(CERT, LIFETIME), at(LATER, "verify"), put(CERT, signed(license_id=OTHER_ID)), at(LATER, "status"),
               put(REVOKED_FILE, f" ,{OTHER_ID}, ,"), at(LATER + 1, "status"),
               put(REVOKED_FILE, f"x,{LIFETIME_ID} ,y"), put(CERT, LIFETIME), at(LATER + 2, "status"),
               at(LATER + 3, "offline", arg=LIFETIME)]},
    {"name": "cancelled trial stays cancelled", "registry": True, "client": {
        "verify": [trial_verdict("revoked", NOW + DAY + 120)]}, "steps": [
        put(CERT, TRIAL), at(NOW + DAY + 120, "verify"), put(CERT, TRIAL), at(NOW + DAY + 130, "status"),
        at(NOW + DAY + 140, "check")]},
    {"name": "forged verify answers change nothing", "client": {"verify": BAD_ANSWERS}, "steps": [
        put(CERT, LIFETIME), put(STATE, '{"last_verify": 1}'), *[at(LATER, "verify") for _ in BAD_ANSWERS],
        at(LATER, "status"), at(LATER, "licensed")]},
    {"name": "a trial needs a signed answer", "client": {"verify": [
        ok({"status": "valid", "server_time": stamp(NOW + DAY)}), trial_verdict("valid", NOW + DAY, nonce="x"),
        trial_verdict("valid", NOW + DAY)]}, "steps": [
        put(CERT, TRIAL), at(NOW + DAY, "verify"), at(NOW + DAY, "check"), at(NOW + DAY, "check"),
        at(NOW + DAY + 1, "status")]},
]
