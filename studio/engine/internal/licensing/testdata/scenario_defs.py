"""The LicenseService scenarios make_scenarios.py runs (also read back by the Go test)."""
import hashlib
import json

from vector_common import OTHER_SEED, license_signer, signed

NOW, DAY = 1_760_000_000, 86400
HWID, OTHER = license_signer.HWID, "cd" * 32
LIFETIME = signed(email="buyer@example.com", license_id="SUBS-2345-6789-ABCD-EFGH")
TRIAL = signed(type="trial", expires_at=NOW + 7 * DAY, license_id="TRIAL-abababababab")
CERT = "home/license.cert"
STATE = "home/license_state.json"
WORKER_DIGEST = hashlib.sha256(b"worker").hexdigest()


def at(now, op, **fields):
    return {"now": now, "op": op, **fields}


def put(path, data):
    return {"op": "write", "path": path, "data": data}


def ok(value):
    return {"return": value}


def fail(message, code="server_error"):
    return {"raise": [message, code]}


SCENARIOS = [
    {"name": "development", "enforce": False, "steps": [
        at(NOW, "status"), at(NOW, "licensed"), at(NOW, "trial"), at(NOW, "offline", arg=LIFETIME),
        at(NOW, "config", arg="https://x.example"), at(NOW, "activate", arg="SUBS-2345-6789-ABCD-EFGH"),
        at(NOW, "verify")]},
    {"name": "no certificate", "steps": [at(NOW, "status"), at(NOW, "licensed")]},
    {"name": "offline lifetime", "steps": [
        at(NOW + 60, "offline", arg=LIFETIME), at(NOW + 3650 * DAY, "status"), at(NOW + 3650 * DAY, "licensed")]},
    {"name": "offline wrapper object", "steps": [
        at(NOW + 60, "offline", arg={"certificate": json.loads(TRIAL)}), at(NOW + DAY + 5, "status")]},
    {"name": "offline refused", "steps": [
        at(NOW, "offline", arg="not json"), at(NOW, "offline", arg="[1]"), at(NOW, "offline", arg=5),
        at(NOW, "offline", arg=signed(OTHER_SEED)), at(NOW, "offline", arg=signed(hwid=OTHER)),
        at(NOW + 8 * DAY, "offline", arg=TRIAL), at(NOW, "offline", arg=signed(version=2)),
        at(NOW, "offline", arg=signed(kid=["x"])), at(NOW, "offline", arg=None), at(NOW, "status")]},
    {"name": "trial countdown", "steps": [
        put(CERT, TRIAL), at(NOW + DAY + 5, "status"), at(NOW + 7 * DAY, "status")]},
    {"name": "clock rollback", "steps": [
        put(CERT, TRIAL), at(NOW + 3 * DAY, "licensed"), at(NOW + DAY, "status")]},
    {"name": "run log rollback", "steps": [
        put(CERT, LIFETIME), put("data/logs/runs.jsonl", '{"kind":"run_started","at":"2026-01-01T00:00:00+00:00"}\n'),
        at(NOW, "status")]},
    {"name": "revoked reason", "steps": [
        put(STATE, '{"revoked_reason": "gone", "last_verify": 5}'), at(NOW, "status"),
        put(STATE, '{"revoked_reason": 7}'), at(NOW, "status"), put(STATE, "[1]"), at(NOW, "status"),
        put(STATE, "{bad"), at(NOW, "status")]},
    {"name": "server address", "steps": [
        put("home/client_config.json", '{"theme": "dark", "server_url": "x", "z": [1, {}]}'),
        at(NOW, "config", arg=" https://lic.example.com// "), at(NOW, "config", arg="ftp://x"),
        at(NOW, "config", arg=True), at(NOW, "config", arg={"a": 1}), at(NOW, "config", arg="http://localhost:8080/"),
        at(NOW, "config", arg=""), at(NOW, "config", arg=None), put("home/client_config.json", "[]"),
        at(NOW, "config", arg="https://y"), at(NOW, "status")]},
    {"name": "online activation", "client": {
        "activate": [ok(LIFETIME), fail("This key is used on another computer.", "bound_elsewhere"),
                     fail("Cannot reach the license server.", "offline"), fail("Odd", "weird"),
                     fail("Odd", None)],
        "transfer": [ok(json.loads(LIFETIME))]}, "steps": [
        at(NOW + 60, "activate", arg=" subs-2345-6789-abcd-efgh "), at(NOW + 60, "activate", arg="SUBS-1111"),
        at(NOW + 60, "activate", arg="SUBS-2345-6789-ABCD-EFGH"), at(NOW + 60, "activate", arg="SUBS-2345-6789-ABCD-EFGH"),
        at(NOW + 60, "activate", arg="SUBS-2345-6789-ABCD-EFGH"), at(NOW + 60, "activate", arg="SUBS-2345-6789-ABCD-EFGH"),
        at(NOW + 90, "activate", arg="SUBS-2345-6789-ABCD-EFGH", move=True)]},
    {"name": "online trial", "client": {
        "trial": [fail("The trial was already used on this computer.", "trial_used"), ok(signed(hwid=OTHER)),
                  ok(json.loads(TRIAL)), ok({"payload": 1})]}, "steps": [
        at(NOW + 60, "trial"), at(NOW + 60, "trial"), at(NOW + 60, "trial"), at(NOW + 60, "trial")]},
    {"name": "verify revoked", "client": {"verify": [ok({"status": "revoked"})]}, "steps": [
        put(CERT, LIFETIME), put(STATE, '{"last_verify": 1760000000}'), at(NOW + 8 * DAY, "verify"),
        at(NOW + 8 * DAY, "status")]},
    {"name": "verify moved", "client": {"verify": [ok({"status": "moved", "extra": 1})]}, "steps": [
        put(CERT, LIFETIME), at(NOW + 8 * DAY, "verify"), at(NOW + 8 * DAY + 1, "status")]},
    {"name": "verify valid refresh", "client": {"verify": [
        ok({"status": "valid", "certificate": signed(email="new@example.com", issued_at=NOW + DAY)}),
        ok({"status": "valid", "certificate": "garbage"})]}, "steps": [
        put(CERT, LIFETIME), at(NOW + 8 * DAY, "verify"), at(NOW + 16 * DAY, "verify"), at(NOW + 17 * DAY, "status")]},
    {"name": "verify not due or not lifetime", "client": {"verify": [fail("down", "offline")]}, "steps": [
        put(CERT, LIFETIME), put(STATE, '{"last_verify": 1760500000}'), at(NOW + 6 * DAY, "verify"),
        at(NOW + 30 * DAY, "verify"), put(CERT, TRIAL), at(NOW + DAY, "verify"),
        put(STATE, '{"last_verify": "x"}'), put(CERT, LIFETIME), at(NOW + 40 * DAY, "verify")]},
    {"name": "legacy machine", "legacy": OTHER, "client": {"verify": [
        ok({"status": "valid", "certificate": signed(hwid=OTHER)})]}, "steps": [
        put(CERT, signed(hwid=OTHER, issued_at=NOW)), at(NOW + 60, "status"), at(NOW + 8 * DAY, "verify"),
        put(CERT, signed(hwid=OTHER, issued_at=1814400001)), at(1814400100, "status")]},
    {"name": "debugger", "debugger": True, "steps": [put(CERT, LIFETIME), at(NOW, "status")]},
    {"name": "tampered worker", "expected": {"studio_ied.exe": "0" * 64}, "steps": [at(NOW, "status")]},
    {"name": "intact worker", "expected": {"studio_ied.exe": WORKER_DIGEST}, "steps": [
        put(CERT, LIFETIME), at(NOW, "status")]},
    {"name": "status cache", "steps": [
        at(NOW, "licensed"), put(CERT, LIFETIME), at(NOW + 30, "licensed"), at(NOW + 61, "licensed"),
        {"op": "remove", "path": CERT}, at(NOW + 50, "licensed"), at(NOW + 200, "licensed")]},
    {"name": "machine id unreadable", "hwid": None, "steps": [at(NOW, "status"), put(CERT, LIFETIME), at(NOW, "licensed")]},
    {"name": "no release keys", "keys": "none", "steps": [put(CERT, LIFETIME), at(NOW, "status")]},
    {"name": "unreadable files", "steps": [
        put(CERT, "﻿" + LIFETIME), at(NOW, "status"), {"op": "write", "path": CERT, "hex": "ff00"},
        at(NOW, "status"), put(CERT, LIFETIME), {"op": "write", "path": "home/.time_anchor", "hex": "39393939393939393939"},
        at(NOW, "status")]},
]
