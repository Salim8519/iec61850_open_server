"""Write generated/answers.json: verify_answer.verified_status verdicts for the Go parity test.

Usage (from the worktree root):
  studio/.venv/Scripts/python.exe -I -B studio/engine/internal/licensing/testdata/make_answers.py
Every case is a server answer (genuine or altered in one way), the request it answers
(nonce, license ID, machine ID) and Python's verdict: a status, or null for "not genuine".
Go must reach the same verdict for each.
"""
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from vector_common import OTHER_SEED, license_signer, signed_raw, write  # noqa: E402
from studio_app.licensing import verify_answer  # noqa: E402

NONCE = "0123456789abcdef" * 2
LICENSE_ID = "SUBS-2345-6789-ABCD-EFGH"
HWID = license_signer.HWID
KID = license_signer.key_id(license_signer.public_key())
GOOD = {"hwid": HWID, "kid": KID, "license_id": LICENSE_ID, "nonce": NONCE, "purpose": "verify-answer",
        "status": "valid", "time": 1_760_000_000, "version": 1}


def answer_of(payload, seed=license_signer.TEST_SEED, signature=None):
    bundle = signed_raw(json.dumps(payload, separators=(",", ":"), sort_keys=True), seed)
    if signature is not None:
        bundle["signature"] = signature
    return {"status": "revoked", "answer": bundle}


def changed(**fields):
    return answer_of({**GOOD, **fields})


def without(name):
    return answer_of({key: value for key, value in GOOD.items() if key != name})


def cases():
    exact = signed_raw('{"hwid": "%s", "kid": "%s", "license_id": "%s", "nonce": "%s", "purpose": "verify-answer", '
                       '"status": "revoked", "time": 1, "version": 1}' % (HWID, KID, LICENSE_ID, NONCE))
    yield "genuine valid", answer_of(GOOD), None
    for status in ("revoked", "moved", "unknown"):
        yield f"genuine {status}", changed(status=status), None
    yield "spaces in the signed text", {"answer": exact}, None
    yield "other signer", answer_of(GOOD, OTHER_SEED), None
    yield "tampered signature", answer_of(GOOD, signature="AAAA" + answer_of(GOOD)["answer"]["signature"][4:]), None
    yield "unreadable signature", answer_of(GOOD, signature="!!"), None
    yield "unknown kid, right signer", changed(kid="deadbeef"), None
    yield "kid as list", changed(kid=["x"]), None
    yield "kid missing", without("kid"), None
    for name in ("hwid", "license_id", "nonce", "purpose", "status", "version"):
        yield f"{name} missing", without(name), None
    yield "wrong nonce", changed(nonce="f" * 32), None
    yield "empty nonce", changed(nonce=""), None
    yield "wrong license", changed(license_id="SUBS-OTHR-OTHR-OTHR-OTHR"), None
    yield "wrong machine", changed(hwid="cd" * 32), None
    yield "wrong purpose", changed(purpose="server-time"), None
    yield "certificate as purpose", changed(purpose="certificate"), None
    for version in (0, 2, 1.0, "1", True, None, [1]):
        yield f"version {json.dumps(version)}", changed(version=version), None
    for status in ("banned", "", "Valid", ["valid"], 1, None, True):
        yield f"status {json.dumps(status)}", changed(status=status), None
    yield "extra fields are fine", changed(extra=[1, 2], time=-5), None
    yield "unsigned only", {"status": "revoked"}, None
    yield "answer null", {"status": "revoked", "answer": None}, None
    yield "answer text", {"status": "revoked", "answer": "revoked"}, None
    yield "answer list", {"status": "revoked", "answer": [1]}, None
    yield "answer without signature", {"answer": {"payload": json.dumps(GOOD)}}, None
    yield "answer without payload", {"answer": {"signature": "AAAA"}}, None
    yield "payload not text", {"answer": {"payload": GOOD, "signature": "AAAA"}}, None
    yield "signature not text", {"answer": {"payload": json.dumps(GOOD), "signature": 5}}, None
    yield "payload not an object", {"answer": signed_raw("[1]")}, None
    yield "payload not json", {"answer": signed_raw("not json")}, None
    yield "wrapped like a certificate", {"answer": {"certificate": answer_of(GOOD)["answer"]}}, None
    yield "not an object", [1], None
    yield "text", "valid", None
    yield "none", None, None
    yield "request license id not text", answer_of(GOOD), {"license_id": 5}
    yield "request license id missing", answer_of(GOOD), {"license_id": None}


def main():
    out = []
    for name, answer, request in cases():
        request = {"nonce": NONCE, "license_id": LICENSE_ID, "hwid": HWID, **(request or {})}
        verdict = verify_answer.verified_status(answer, request["nonce"], request["license_id"], request["hwid"],
                                                license_signer.keys())
        out.append({"name": name, "answer": answer, "request": request, "status": verdict})
    write("answers.json", {"cases": out})
    print(f"{len(out)} cases, {sum(1 for case in out if case['status'])} genuine")


if __name__ == "__main__":
    main()
