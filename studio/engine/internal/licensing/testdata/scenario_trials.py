"""Trial-hardening scenarios (ADR-0046 addendum 2026-10-19) for make_scenarios.py.

Signed server time in three places, rollback after deleted anchor copies, the 48 h drift
allowance, the 3-day missing-anchor grace with "Check online now", the email step and
stamps that must never pass as licenses. "registry": True gives the service an in-memory
HKCU\\Software\\ScadaBench; its values are recorded after every step.
"""
import json

from scenario_defs import CERT, DAY, NOW, STATE, TRIAL, at, fail, ok, put
from vector_common import license_signer

ANCHOR, STAMP_FILE = "home/.time_anchor", "home/.server_time"


def stamp(moment):
    return license_signer.stamp(moment)


def trial_answer(moment):
    return ok({"certificate": json.loads(TRIAL), "server_time": stamp(moment)})


TRIAL_SCENARIOS = [
    {"name": "trial keeps the signed time", "registry": True, "client": {
        "trial": [trial_answer(NOW + 60)],
        "verify": [ok({"status": "valid", "server_time": stamp(NOW + DAY + 120)}), fail("down", "offline"),
                   ok({"status": "valid"}), ok([1])]}, "steps": [
        at(NOW + 60, "trial"), at(NOW + 120, "verify", trial_only=True), at(NOW + DAY + 120, "verify"),
        at(NOW + DAY + 130, "status"), at(NOW + 3 * DAY, "verify", trial_only=True), at(NOW + 3 * DAY, "check"),
        at(NOW + 3 * DAY, "check"), at(NOW + 3 * DAY, "status")]},
    {"name": "rollback after the anchor file is deleted", "registry": True, "steps": [
        put(CERT, TRIAL), at(NOW + 5 * DAY, "status"), {"op": "remove", "path": ANCHOR},
        at(NOW + 2 * DAY, "status"), at(NOW + 5 * DAY - 47 * 3600, "status"), at(NOW + 5 * DAY, "licensed")]},
    {"name": "signed time catches a rollback without anchors", "client": {"trial": [trial_answer(NOW + 5 * DAY)]},
     "steps": [
        at(NOW + 5 * DAY, "trial"), {"op": "remove", "path": ANCHOR},
        put(STATE, '{"revoked_reason": null, "last_verify": 1760432000}'), at(NOW + 2 * DAY, "status"),
        at(NOW + 4 * DAY, "status")]},
    {"name": "missing anchor asks one online check", "client": {
        "trial": [trial_answer(NOW + DAY)],
        "verify": [fail("down", "offline"), fail("down", "offline"),
                   ok({"status": "valid", "server_time": stamp(NOW + 4 * DAY + 60)})]}, "steps": [
        at(NOW + DAY, "trial"), {"op": "remove", "path": ANCHOR}, {"op": "remove", "path": STAMP_FILE},
        put(STATE, '{"last_verify": 1760086400}'), at(NOW + DAY + 61, "licensed"), at(NOW + 4 * DAY - 60, "status"),
        at(NOW + 4 * DAY + 120, "status"), at(NOW + 4 * DAY + 120, "verify", trial_only=True),
        at(NOW + 4 * DAY + 120, "check"), at(NOW + 4 * DAY + 120, "check"), at(NOW + 4 * DAY + 70, "status"),
        at(NOW + 6 * DAY, "licensed")]},
    {"name": "server time past the end ends the trial", "registry": True, "client": {
        "verify": [ok({"status": "valid", "server_time": stamp(NOW + 7 * DAY + 3600)})]}, "steps": [
        put(CERT, TRIAL), at(NOW + 7 * DAY - 3600, "verify"), at(NOW + 7 * DAY - 3600, "status"),
        at(NOW + 7 * DAY - 3600, "verify")]},
    {"name": "older build trial is never asked", "client": {"verify": [fail("down", "offline")]}, "steps": [
        put(CERT, TRIAL), put(ANCHOR, str(NOW + 600)), at(NOW + DAY, "licensed"), at(NOW + 6 * DAY, "status"),
        at(NOW + 6 * DAY, "verify", trial_only=True)]},
    {"name": "email step", "client": {
        "trial": [fail("Please confirm your email address.", "email_required"), trial_answer(NOW + 60)],
        "email-code": [ok({"status": "sent"}), ok({}), ok({"status": "no_code_needed"}), ok([1]),
                       fail("x", "bad_email")]}, "steps": [
        at(NOW + 60, "trial"), at(NOW + 60, "email_code", arg="not an address"), at(NOW + 60, "email_code", arg=5),
        at(NOW + 60, "email_code", arg=" me@company.com "), at(NOW + 60, "email_code", arg="me@company.com"),
        at(NOW + 60, "email_code", arg=["a@b"]), at(NOW + 60, "email_code", arg="a@" + "b" * 253),
        at(NOW + 60, "email_code", arg="a@b"), at(NOW + 60, "email_code", arg="a@b"),
        at(NOW + 60, "trial", email="me@company.com", code=123456), at(NOW + 60, "status")]},
    {"name": "stamps never pass as licenses", "registry": True, "steps": [
        at(NOW, "offline", arg=json.dumps(stamp(NOW))), put(CERT, TRIAL),
        put(STAMP_FILE, json.dumps(json.loads(TRIAL))), at(NOW + DAY, "status"),
        put(STAMP_FILE, json.dumps(stamp(NOW + 9 * DAY)).replace(str(NOW + 9 * DAY), str(NOW + 99 * DAY))),
        at(NOW + DAY + 1, "status"), put(STAMP_FILE, "garbage"), at(NOW + DAY + 2, "licensed")]},
    {"name": "lifetime never needs the daily check", "client": {"verify": [fail("down", "offline")]}, "steps": [
        put(CERT, json.dumps(json.loads(license_signer.bundle()))), at(NOW + 3650 * DAY, "verify", trial_only=True),
        at(NOW + 3650 * DAY, "check"), at(NOW + 3650 * DAY - 301, "status")]},
]
