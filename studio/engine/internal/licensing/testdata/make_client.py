"""Write generated/client.json: LicenseClient requests and outcomes for canned server answers.

Usage (from the worktree root):
  studio/.venv/Scripts/python.exe -I -B studio/engine/internal/licensing/testdata/make_client.py
No network is used: a fake opener answers. The Go test replays the same answers from an
httptest server and must send the same request bodies and reach the same outcome.
"""
import io
import json
import sys
from pathlib import Path
from urllib.error import HTTPError, URLError

sys.path.insert(0, str(Path(__file__).resolve().parent))
from vector_common import outcome, signed, write  # noqa: E402
from studio_app.licensing.client import LicenseClient  # noqa: E402

HWID = "ab" * 32
KEY = "SUBS-2345-6789-ABCD-EFGH"
LOCAL = "http://127.0.0.1:1"
CHALLENGE = {"status": 200, "body": '{"nonce": "n-61850", "timestamp": 1760000000, "difficulty": 8, "mac": "m1"}'}


def reply(body, status=200):
    return {"status": status, "body": body if isinstance(body, str) else json.dumps(body)}


CASES = [
    ("trial", LOCAL, [CHALLENGE, reply({"certificate": signed()})]),
    ("activate", LOCAL + "//", [CHALLENGE, reply({"certificate": json.loads(signed())})]),
    ("transfer", LOCAL, [CHALLENGE, reply({"certificate": "x", "extra": [1]})]),
    ("verify", LOCAL, [reply({"status": "valid"})]),
    ("verify_number_key", LOCAL, [reply({"status": "revoked"})]),
    ("activate", LOCAL, [CHALLENGE, reply({"error": "Key used elsewhere.", "code": "bound_elsewhere"}, 409)]),
    ("activate", LOCAL, [CHALLENGE, reply("<html>oops</html>", 500)]),
    ("trial", LOCAL, [CHALLENGE, reply({"error": "", "code": "x"}, 400)]),
    ("trial", LOCAL, [CHALLENGE, reply({"code": None}, 404)]),
    ("trial", LOCAL, [CHALLENGE, reply({"error": 5}, 429)]),
    ("trial", LOCAL, [CHALLENGE, reply("[1]", 403)]),
    ("trial", LOCAL, [{"urlerror": True}]),
    ("trial", LOCAL, [CHALLENGE, reply("{bad")]),
    ("trial", LOCAL, [reply({"nonce": "n", "timestamp": 1, "difficulty": 0, "mac": "m"})]),
    ("trial", LOCAL, [reply({"nonce": "n", "timestamp": 1, "difficulty": 27})]),
    ("trial", LOCAL, [reply({"nonce": "n", "difficulty": 4})]),
    ("trial", LOCAL, [reply({"nonce": "n", "timestamp": 1, "difficulty": "6", "mac": None}),
                      reply({"certificate": "c"})]),
    ("trial", LOCAL, [CHALLENGE, reply({"no": "certificate"})]),
    ("trial", LOCAL, [CHALLENGE, reply("﻿" + json.dumps({"certificate": "bom"}))]),
    ("trial", "ftp://license.example", []),
    ("trial", "", []),
    ("verify", "http://localhost:1", [reply("[]")]),
    ("trial_parts", LOCAL, [CHALLENGE, reply({"certificate": json.loads(signed()), "server_time": {"payload": "p"}})]),
    ("trial_empty_parts", LOCAL, [CHALLENGE, reply({"certificate": "c"})]),
    ("trial_email", LOCAL, [CHALLENGE, reply({"certificate": "c"})]),
    ("trial_email_number", LOCAL, [CHALLENGE, reply({"certificate": "c"})]),
    ("trial_email_nocode", LOCAL, [CHALLENGE, reply({"certificate": "c"})]),
    ("trial_no_email", LOCAL, [CHALLENGE, reply({"certificate": "c"})]),
    ("email_code", LOCAL, [CHALLENGE, reply({"status": "sent"})]),
    ("email_code", LOCAL, [CHALLENGE, reply([1])]),
    ("trial", LOCAL, [CHALLENGE, reply({"error": "terse", "code": "trial_used"}, 409)]),
    ("trial", LOCAL, [CHALLENGE, reply({"error": "terse", "code": "email_required"}, 409)]),
    ("trial", LOCAL, [CHALLENGE, reply({"code": "bad_code"}, 400)]),
    ("trial", LOCAL, [CHALLENGE, reply({"error": "x", "code": "email_disposable"}, 400)]),
    ("email_code", LOCAL, [CHALLENGE, reply({"error": "x", "code": "bad_email"}, 400)]),
    ("trial", LOCAL, [CHALLENGE, reply({"error": "x", "code": [1]}, 400)]),
    ("verify", LOCAL, [reply({"error": "x", "code": "email_used"}, 409)]),
    ("verify", LOCAL, [reply({"error": "x", "code": 5}, 409)]),
]
PARTS = {"guid": "g", "volume": "", "firmware": "f"}


class Response(io.BytesIO):
    pass


def opener_for(answers, requests):
    def opener(request, timeout):
        requests.append({"method": request.get_method(), "path": request.full_url.split("/", 3)[3],
                         "body": request.data.decode("utf-8") if request.data else None,
                         "content_type": request.headers.get("Content-type"),
                         "user_agent": request.headers.get("User-agent"), "timeout": timeout})
        answer = answers.pop(0)
        if answer.get("urlerror"):
            raise URLError("no route")
        body = answer["body"].encode("utf-8")
        if answer["status"] >= 400:
            raise HTTPError(request.full_url, answer["status"], "error", {}, io.BytesIO(body))
        return Response(body)
    return opener


def run(name, url, answers):
    requests = []
    client = LicenseClient(url, opener=opener_for(list(answers), requests))
    calls = {"trial": lambda: client.start_trial(HWID), "activate": lambda: client.activate(KEY, HWID),
             "transfer": lambda: client.transfer(KEY, HWID), "verify": lambda: client.verify(KEY, HWID),
             "verify_number_key": lambda: client.verify(5, HWID),
             "trial_parts": lambda: client.start_trial(HWID, PARTS),
             "trial_empty_parts": lambda: client.start_trial(HWID, {"guid": "", "volume": "", "firmware": ""}),
             "trial_email": lambda: client.start_trial(HWID, {}, "me@company.com", " 123456 "),
             "trial_email_number": lambda: client.start_trial(HWID, None, "me@company.com", 123456),
             "trial_email_nocode": lambda: client.start_trial(HWID, None, "me@company.com", None),
             "trial_no_email": lambda: client.start_trial(HWID, None, "", "123"),
             "email_code": lambda: client.request_trial_code(HWID, "me@company.com")}
    result = outcome(calls[name])
    if "ok" in result:
        result["ok"] = json.dumps(result["ok"], separators=(",", ":"))
    return {"call": name, "url": url, "answers": answers, "requests": requests, **result}


def main():
    write("client.json", {"hwid": HWID, "key": KEY, "cases": [run(*case) for case in CASES]})


if __name__ == "__main__":
    main()
