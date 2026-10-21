"""Write generated/vectors.json: the frozen Python licensing answers for the Go unit tests.

Usage (from the worktree root):
  studio/.venv/Scripts/python.exe -I -B studio/engine/internal/licensing/testdata/make_vectors.py
Also run make_scenarios.py, make_client.py and make_trial.py. The Go tests need only the JSON files.
"""
import base64
import random
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from vector_common import OTHER_SEED, compact, license_signer, outcome, signed, signed_raw, write  # noqa: E402
from vector_iso import anchor_cases, iso_cases  # noqa: E402
from studio_app.domain import license_policy, trial_policy  # noqa: E402
from studio_app.licensing import certificate, client, ed25519_verify, hwid, release_key  # noqa: E402
from studio_app.licensing.service import LicenseService, mask_key  # noqa: E402

HWID = license_signer.HWID
NOW = 1_760_000_000


def b64_cases():
    texts = ["", "QQ", "QQ=", "QQ==", "QQ===", "Q", "QUJD", "QUJDRA", " QUJDRA ", "QU JD\nRA", "QU=JD",
             "-_-_", "+/+/", "QUJD!", "QUJDRA=\t", "Q=Q=", "QUI=x", "é", "　QUJD　", "QUJ"]
    texts.append(license_signer.bundle().split('"signature": "')[1].rstrip('"}'))
    cases = []
    for text in texts:
        entry = {"text": text}
        stripped = outcome(lambda: certificate._decode_signature(text).hex())
        entry["strip"] = stripped.get("ok")
        plain = outcome(lambda: base64.urlsafe_b64decode(text + "=" * (-len(text) % 4)).hex())
        entry["pad"] = plain.get("ok")
        cases.append(entry)
    return cases


def ed25519_cases():
    rng = random.Random(25519)
    public = license_signer.public_key()
    message = b"scadabench"
    good = license_signer.sign(message)
    order = ed25519_verify.Q.to_bytes(32, "little")
    p_bytes = (ed25519_verify.P + 1).to_bytes(32, "little")  # y = 1 encoded as y + p
    cases = [(public, message, good), (public, message + b"!", good), (public, message, good[:32] + order),
             (public[:31], message, good), (public, message, good[:63]), (p_bytes, message, good),
             ((1).to_bytes(32, "little"), message, good), (((1 << 255) | 1).to_bytes(32, "little"), message, good),
             ((ed25519_verify.P - 1 | 1 << 255).to_bytes(32, "little"), message, good)]
    for _ in range(40):
        flipped = bytearray(good)
        flipped[rng.randrange(64)] ^= 1 << rng.randrange(8)
        cases.append((public, message, bytes(flipped)))
        cases.append((bytes(rng.randrange(256) for _ in range(32)), message, good))
    return [{"public": p.hex(), "message": m.hex(), "signature": s.hex(), "ok": ed25519_verify.verify(p, m, s)}
            for p, m, s in cases]


def certificate_cases():
    good = signed()
    texts = [good, signed(type="trial", expires_at=NOW + 86400), '{"certificate": ' + good + "}", "not json",
             "[1]", '{"payload": 1, "signature": "x"}', '{"payload": "x", "signature": "AA"}',
             compact(signed_raw("[1]")), compact(signed_raw('{"a":1')), good.replace('"signature": "', '"signature": "!'),
             good.replace('"signature": "', '"signature": "A'), signed(OTHER_SEED), signed(kid="00000000"),
             signed(kid=["x"]), signed(kid=None), "﻿" + good, good.replace("lifetime", "trial"),
             compact(signed_raw("﻿" + compact({"version": 1}))), '{"certificate": [], "payload": "{}", "signature": ""}',
             "  " + good + "\n"]
    cases = []
    for keys_name, keys in (("test", license_signer.keys()), ("none", {}), ("release", None)):
        for text in texts if keys_name == "test" else texts[:2]:
            result = outcome(lambda: certificate.verify_bundle(certificate.parse_bundle(text), keys))
            if "ok" in result:
                result["ok"] = compact(result["ok"])
            cases.append({"text": text, "keys": keys_name, **result})
    return cases


def policy_cases():
    base = {"version": 1, "type": "lifetime", "hwid": HWID, "issued_at": NOW, "license_id": "SUBS-AAAA"}
    variants = [None, base, {**base, "version": 2}, {**base, "version": "1"}, {**base, "version": 1.9},
                {**base, "version": float("nan")}, {**base, "version": None}, {**base, "type": "pro"},
                {**base, "type": 5}, {**base, "hwid": "cd" * 32}, {**base, "issued_at": NOW + 3600},
                {**base, "issued_at": "x"}, {**base, "issued_at": float("inf")}, {**base, "expires_at": NOW + 1},
                {**base, "type": "trial", "expires_at": NOW + 7 * 86400}, {**base, "type": "trial", "expires_at": NOW},
                {**base, "expires_at": NOW - 5}, {**base, "email": "a@b"}, {**base, "email": ""},
                {**base, "license_id": 7}, {k: v for k, v in base.items() if k != "issued_at"},
                {k: v for k, v in base.items() if k != "type"}, {**base, "expires_at": " 1_760_000_100 "},
                {**base, "version": True}, {**base, "hwid": [HWID]}]
    cases = []
    for payload in variants:
        for now, max_seen, dev in ((NOW + 60, 0, False), (NOW - 400, 0, False), (NOW + 60, NOW + 1000, False),
                                   (NOW + 3 * 86400 + 5, 0, False), (NOW, 0, True), (NOW, NOW + 47 * 3600, False),
                                   (NOW, NOW + 49 * 3600, False)):
            for tolerance in (license_policy.CLOCK_TOLERANCE_SECONDS, trial_policy.TRIAL_CLOCK_TOLERANCE_SECONDS):
                result = outcome(lambda: license_policy.evaluate_certificate(
                    payload, HWID, now, max_seen, dev, tolerance).as_dict())
                if "ok" in result:
                    result["ok"] = compact(result["ok"])
                cases.append({"payload": None if payload is None else compact(payload), "now": now,
                              "max_seen": max_seen, "dev": dev, "tolerance": tolerance, **result})
    return cases


def key_cases():
    inputs = ["SUBS-2345-6789-ABCD-EFGH", " subs-2345-6789-abcd-efgh ", "SUBS-2345-6789-ABCD-EFG1",
              "SUBS-2345 6789-ABCD-EFGH", "SUBS-ßßßß-2345-6789-ABCD", "SUBS-2345-6789-ABCD-EFGH\n",
              "", None, True, 5, "SUBS-2345-6789-ABCD-EFGH-JKMN", "SUBS-IIII-2345-6789-ABCD", "SUBS-ﬀﬀ-2345-6789-ABCD"]
    normalized = [{"input": compact(value), **outcome(lambda: LicenseService.normalize_key(value))} for value in inputs]
    masks = [{"input": compact(value), "output": compact(mask_key(value))}
             for value in ["SUBS-2345-6789-ABCD-EFGH", "SUBS-", "SUBS-x", "TRIAL-abc", "", None, 0, 7, "subs-AAAA-1234"]]
    return normalized, masks


def hwid_cases():
    rng = random.Random(1814400000)
    composed = []
    for guid, serial, uuid in [("ABCDEF01-2345-6789-abcd-ef0123456789", 0x1234ABCD, "00112233445566778899aabbccddeeff"),
                               ("  Guid\t", 0, ""), ("x", 0xFFFFFFFF, "abc")]:
        composed.append({"guid": guid, "serial": serial, "uuid": uuid, "v2": hwid.compose(guid, serial, uuid),
                         "v1": hwid.compose_legacy(guid, serial)})
    bound = []
    for cert, issued, current, legacy in [("L", 1, "C", "L"), ("L", hwid.LEGACY_ISSUED_BEFORE, "C", "L"), ("C", 1, "C", "L"),
                                          ("L", 1, "C", None), ("L", "12", "C", "L"), ("L", "x", "C", "L"), (5, 1, "C", "L"),
                                          ("L", 1.5, "C", "L"), ("L", None, "C", "L"), ("L", 1, "L", "L")]:
        bound.append({"cert": compact(cert), "issued": compact(issued), "current": current, "legacy": legacy,
                      **outcome(lambda: hwid.bound_hwid(cert, issued, current, legacy))})
    tables = [b"", b"\0" * 8, bytes(8) + bytes([1, 27, 0, 0]) + bytes(range(4, 27)) + b"\0\0",
              bytes(8) + bytes([0, 4, 0, 0]) + b"abc\0\0" + bytes([1, 25, 0, 0]) + bytes(range(21)) + b"\0\0",
              bytes(8) + bytes([1, 3, 0, 0]) + bytes(30), bytes(8) + bytes([1, 25, 0, 0]) + bytes(10)]
    for _ in range(30):
        tables.append(bytes(8) + bytes(rng.choice([0, 1, 1, 2, 3, 4, 24, 30, 0xff]) for _ in range(rng.randint(0, 80))))
    smbios = [{"raw": raw.hex(), "uuid": hwid._smbios_system_uuid(raw)} for raw in tables]
    return composed, bound, smbios


def main():
    normalized, masks = key_cases()
    composed, bound, smbios = hwid_cases()
    test_public = license_signer.public_key()
    write("vectors.json", {
        "release_keys": {kid: key.hex() for kid, key in release_key.public_keys().items()},
        "test_key": {"kid": release_key.key_id(test_public), "public": test_public.hex()},
        "default_server_url": release_key.DEFAULT_SERVER_URL, "base64": b64_cases(),
        "ed25519": ed25519_cases(), "certificates": certificate_cases(), "policy": policy_cases(),
        "normalize_key": normalized, "mask_key": masks, "compose": composed, "bound_hwid": bound,
        "smbios": smbios, "iso": iso_cases(), "anchor": anchor_cases(),
        "solve": [{"nonce": nonce, "difficulty": difficulty, **outcome(lambda: client.solve(nonce, difficulty))}
                  for nonce, difficulty in [("abc", 8), ("n-1", 12), ("", 1), ("x", 0), ("x", 27)]],
    })


if __name__ == "__main__":
    main()
