"""License and update-check steps for http_gen.py (test-signed certificates and releases only)."""
import json

import license_signer
from test_update_check import RECORD, answer

NOW, DAY = 1_760_000_000, 86400
LIFETIME = license_signer.bundle(email="buyer@example.com", license_id="SUBS-2345-6789-ABCD-EFGH")
TRIAL = license_signer.bundle(type="trial", expires_at=NOW + 7 * DAY, license_id="TRIAL-abababababab")
OTHER_HWID = license_signer.bundle(hwid="cd" * 32)
GOOD = answer().decode()
IMPORTANT = answer({**RECORD, "min_version": "0.1.14"}).decode()
NO_BODY = object()
NONCE = "0123456789abcdef" * 2  # the service's nonce provider: signed canned answers can name it


def req(method, path, body=NO_BODY, raw=None, ctype=None):
    step = {"op": "request", "method": method, "path": "/api/v1" + path}
    if body is not NO_BODY:
        step["json"] = body
    if raw is not None:
        step["body"] = raw
        if ctype:
            step["content_type"] = ctype
    return step


def op(name, **fields):
    return {"op": name, **fields}


def ok(value):
    return {"return": value}


def fail(message, code="server_error"):
    return {"raise": [message, code]}


def license_steps():
    act, offline, config = "/license/activate", "/license/activate-offline", "/license/config"
    return [
        op("license", enforce=False), op("now", value=NOW),
        req("GET", "/license/status"), req("POST", "/license/trial"), req("POST", offline, {"certificate": LIFETIME}),
        req("POST", "/license/check-online"), req("POST", "/license/trial/email-code", {"email": "a@b"}),
        req("PUT", config, {"server_url": "https://x.example"}), req("POST", act, {"license_key": "SUBS-2345"}),
        op("license", enforce=True, test_server=True),  # the test-only seam keeps the old address rules
        req("GET", "/license/status"), req("GET", "/license/status?x=1"),
        op("client", answers={"trial": [fail("The trial was already used on this computer.", "trial_used"),
                                        fail("Cannot reach the license server.", "offline"),
                                        fail("Server trouble"), fail("Odd", "weird"), fail("Odd", None),
                                        ok(OTHER_HWID), ok({"payload": 1}), ok(json.loads(TRIAL))]}),
        *[req("POST", "/license/trial") for _ in range(7)],
        req("POST", "/license/trial", raw="not json", ctype="text/plain"),
        req("GET", "/license/status"),
        *trial_email_steps(),
        op("now", value=NOW + 61), req("GET", "/license/status"),
        op("client", answers={"activate": [ok(LIFETIME), fail("This key is used on another computer.", "bound_elsewhere"),
                                           fail("No such key.", "unknown_key")],
                              }),
        req("POST", act), req("POST", act, [1]), req("POST", act, "SUBS-2345-6789-ABCD-EFGH"),
        req("POST", act, {"license_key": 5}), req("POST", act, {"license_key": None}),
        req("POST", act, {"license_key": "SUBS-1111"}), req("POST", act, raw="{bad", ctype="application/json"),
        req("POST", act, {"license_key": " subs-2345-6789-abcd-efgh "}),
        req("POST", act, {"license_key": "SUBS-2345-6789-ABCD-EFGH", "move": 0}),
        req("POST", act, {"license_key": "SUBS-2345-6789-ABCD-EFGH", "move": "yes"}),
        req("POST", act, {"license_key": "SUBS-2345-6789-ABCD-EFGH", "move": []}),
        req("POST", act, {"license_key": "SUBS-2345-6789-ABCD-EFGH", "move": True}),
        req("POST", act, {"move": True}), req("POST", act, {"license_key": "bad", "move": 1}),
        req("POST", act, {"license_key": "SUBS-2345-6789-ABCD-EFGH"}),
        op("now", value=NOW + 200), req("GET", "/license/status"),
        req("POST", offline), req("POST", offline, {"certificate": "not json"}), req("POST", offline, {"certificate": 5}),
        req("POST", offline, {"certificate": None}), req("POST", offline, {"certificate": "[1]"}),
        req("POST", offline, {"certificate": OTHER_HWID}), req("POST", offline, [LIFETIME]),
        req("POST", offline, {"certificate": {"certificate": json.loads(TRIAL)}}),
        req("POST", offline, {"certificate": LIFETIME}),
        req("PUT", config, {"server_url": " https://lic.example.com// "}), req("PUT", config, {"server_url": "ftp://x"}),
        req("PUT", config, {"server_url": True}), req("PUT", config, {"server_url": {"a": 1}}),
        req("PUT", config, {"server_url": "http://localhost:8080/"}), req("PUT", config, raw="x", ctype="text/plain"),
        req("PUT", config, ["https://y"]), req("PUT", config, {"server_url": "https://y"}),
        op("now", value=NOW + 400), req("GET", "/license/status"),
        op("license_file", path="home/license.cert", data=TRIAL), op("now", value=NOW + 3 * DAY),
        req("GET", "/license/status"), op("now", value=NOW + DAY), req("GET", "/license/status"),
        op("license_file", path="home/license_state.json", data='{"revoked_reason": "gone"}'),
        op("license_file", path="home/license.cert", hex="ff"), op("now", value=NOW + 9 * DAY),
        req("GET", "/license/status"),
        op("license_file", path="home/client_config.json", data="[]"), op("now", value=NOW + 10 * DAY),
        req("GET", "/license/status"), req("PUT", config, {"server_url": "https://z"}),
        *packaged_steps(),
    ]


def packaged_steps():
    """A packaged service without the test seam: the built-in server only, no moves (ADR-0097)."""
    act, offline, config = "/license/activate", "/license/activate-offline", "/license/config"
    return [
        op("license", enforce=True), op("now", value=NOW + DAY),
        op("license_file", path="home/client_config.json", data='{"server_url": "http://127.0.0.1:9"}'),
        req("GET", "/license/status"),
        req("PUT", config, {"server_url": "http://127.0.0.1:9"}), req("PUT", config, {"server_url": "https://z"}),
        req("PUT", config, {"server_url": ""}), req("PUT", config, raw="x", ctype="text/plain"), req("PUT", config, [1]),
        op("client", answers={"activate": [ok(LIFETIME)]}),
        req("POST", act, {"license_key": "SUBS-2345-6789-ABCD-EFGH", "move": True}),
        req("POST", act, {"license_key": "SUBS-2345-6789-ABCD-EFGH", "move": False}), req("GET", "/license/status"),
        op("license", enforce=False), req("PUT", config, {"server_url": "https://z"}),
        req("POST", act, {"license_key": "SUBS-2345-6789-ABCD-EFGH", "move": True}),
    ]


def signed_answer(status, nonce=NONCE, license_id="TRIAL-abababababab", hwid=license_signer.HWID):
    """The server's signed verify answer for the trial the steps install."""
    return license_signer.verify_answer(nonce, license_id, hwid, status)


def trial_email_steps():
    """The email step and "Check online now" (trial hardening, change 0106)."""
    code, check = "/license/trial/email-code", "/license/check-online"
    return [
        op("client", answers={"trial": [fail("Please confirm your email address.", "email_required"),
                                        fail("Used.", "email_used"), fail("Bad code.", "bad_code")],
                              "email-code": [ok({"status": "sent"}), fail("Mail down"), ok({}), ok([1])]}),
        req("POST", "/license/trial", {"email": "me@company.com", "code": 123456}),
        req("POST", "/license/trial", {"email": "", "code": "1"}), req("POST", "/license/trial", [1]),
        req("POST", code, {"email": "me@company.com"}), req("POST", code), req("POST", code, {"email": 5}),
        req("POST", code, raw="x", ctype="text/plain"), req("POST", code, {"email": "a@b"}),
        req("POST", code, {"email": " a@b "}), req("POST", code, {"email": "a@b"}),
        req("POST", check), req("POST", check, {"ignored": True}),
        op("client", answers={"verify": [
            ok({"status": "valid"}),  # unsigned: treated like an unreachable server
            ok({"status": "valid", "answer": signed_answer("valid")}),  # signed but no server time
            ok({"status": "revoked", "answer": signed_answer("valid"), "server_time": license_signer.stamp(NOW + 30)}),
            ok({"status": "valid", "answer": signed_answer("valid", nonce="f" * 32),
                "server_time": license_signer.stamp(NOW + 30)})]}),
        req("POST", check), req("POST", check), req("POST", check), req("POST", check), req("GET", "/license/status"),
    ]


def update_steps():
    path, check = "/app/update", "/app/update/check"
    return [
        op("now", value=1_000_000), req("GET", path), op("answers", queue=[{"body": GOOD, "etag": '"e1"'}]),
        req("POST", check), req("POST", check), req("POST", check, {"ignored": True}),
        op("now", value=1_000_120), op("answers", queue=[{"http": 304}]), req("POST", check),
        req("PUT", path, {"dismissed": "0.1.14"}), req("PUT", path, {"auto": False}), req("PUT", path, {"auto": "yes"}),
        req("PUT", path, {"other": 1}), req("PUT", path), req("PUT", path, [1]), req("PUT", path, {}),
        req("PUT", path, {"dismissed": "x"}), req("PUT", path, {"dismissed": 5}), req("PUT", path, {"dismissed": 0}),
        req("PUT", path, {"dismissed": None, "auto": True}), req("PUT", path, raw="{", ctype="application/json"),
        req("PUT", path, raw="x", ctype="text/plain"), req("PUT", path, None), req("PUT", path, "auto"),
        req("GET", path),
        op("now", value=2_000_000), op("answers", queue=[{"urlerror": True}]), req("POST", check),
        op("now", value=2_000_100), op("answers", queue=[{"http": 404}]), req("POST", check),
        op("now", value=2_000_200), op("answers", queue=[{"http": 500}]), req("POST", check),
        op("now", value=2_000_300), op("answers", queue=[{"body": "[1]", "etag": None}]), req("POST", check),
        op("now", value=2_000_400), op("answers", queue=[{"body": IMPORTANT, "etag": '"e2"'}]), req("POST", check),
        req("PUT", path, {"dismissed": "0.1.14"}), req("GET", path),
        op("update_file", data='{"release": {"date": "x"}}'), req("GET", path),
        op("update_file", data="[1]"), req("GET", path),
        op("update_file", data='{"checked_at": "x"}'), req("POST", check),
        op("update_file", data='{"release": {"version": "1.0.0", "min_version": 3}}'), req("GET", path),
    ]
