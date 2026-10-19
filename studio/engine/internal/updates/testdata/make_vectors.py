"""Write generated/vectors.json: the frozen Python update-check answers for the Go tests.

Usage (from the worktree root):
  studio/.venv/Scripts/python.exe -I -B studio/engine/internal/updates/testdata/make_vectors.py
Release answers are signed with the test key of studio/tests/unit/license_signer.py the way
studio/tests/unit/test_update_check.py `answer()` signs them. No network is used.
"""
import base64
import json
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
STUDIO = HERE.parents[3]
sys.path[:0] = [str(HERE), str(STUDIO / "backend"), str(STUDIO / "tests" / "unit")]

from license_signer import keys, public_key  # noqa: E402
from test_update_check import RECORD, answer  # noqa: E402
from studio_app.licensing.release_key import key_id  # noqa: E402
from studio_app.updates import versions  # noqa: E402
from studio_app.updates.service import UpdateService  # noqa: E402
from update_scenarios import run_scenarios  # noqa: E402

VERSIONS = ["0.1.13", "0.1.14", "0.1.13+dev", "0.10.2", "1.2", "1.2.3.4", "01.002.0003", "12345.1.1",
            "0.1.14\n", "0.1.14+dev\n", "0.1.14 ", "٠.١.٢", "0.1.14+", "0.1.14+a_b",
            "", None, 0, False, 5, 1.5, ["0.1.14"], "v0.1.14", "0.1.14\n\n", "１.２.３"]
URLS = ["https://scadabench.com/download", "https://www.scadabench.com/x", "https://download.scadabench.com",
        "http://scadabench.com/x", "https://evil.example/x", "https://SCADABENCH.COM/x", "HTTPS://scadabench.com",
        "https://scadabench.com:443/x", "https://scadabench.com:0/x", "https://scadabench.com:/x",
        "https://scadabench.com:00/x", "https://scadabench.com:99999/x", "https://scadabench.com:x/",
        "https://user@scadabench.com/", "https://@scadabench.com/", "https://:pw@scadabench.com/",
        "https://:pé@scadabench.com/", "https://:p：@scadabench.com/", "https://[::1]/",
        "https://scadabench.com]/", " \x01https://scadabench.com/", "ht\ttps://scadabench.com/",
        "https:/\n/scadabench.com/", "https://scadabench.com%41/", "https://scadabench.com.evil/",
        "https://evil.com@scadabench.com/", "https://scadabench.com@evil.com/", "https:scadabench.com",
        "//scadabench.com/x", "https://scadabench.com?x", "https://scadabench.com#x", "https://scаdabench.com/",
        "https://scadabench.com\\@evil/", "", None, 5, "https://Kadabench.com/"]


def safe(url):
    try:
        return versions._safe_url(url)
    except ValueError:
        return False


def outcome(action):
    try:
        return {"ok": action()}
    except Exception as error:  # noqa: BLE001 - the class is the vector
        return {"error": type(error).__name__}


def records():
    base = dict(RECORD)
    portable = {"url": "https://download.scadabench.com/p.exe", "sha256": "b" * 64}
    variants = [base, {**base, "min_version": "0.1.14"}, {**base, "min_version": None}, {**base, "min_version": "x"},
                {**base, "version": "0.1.14+dev"}, {**base, "version": 5}, {**base, "date": "2026-13-45"},
                {**base, "date": "1999-01-01"}, {**base, "notes": "  padded\n"}, {**base, "notes": "x" * 601},
                {**base, "notes": "é" * 600}, {**base, "notes": None}, {**base, "page": "https://evil.example"},
                {**base, "installer": None}, {**base, "portable": portable}, {**base, "portable": None},
                {**base, "portable": {"url": portable["url"], "sha256": "B" * 64}},
                {**base, "installer": {**base["installer"], "sha256": "a" * 64 + "\n"}},
                {**base, "extra": 1}, {k: v for k, v in base.items() if k != "notes"}, [1], "x",
                {**base, "min_version": 7}, {**base, "date": "2026-10-20\n"}]
    return variants


def verify_cases(service):
    good = answer()
    bodies = [good, answer({**RECORD, "version": "0.1.15"}), answer(domain=b""), b"{bad", b"[1]",
              b'{"payload": 1, "signature": "x"}', b"\xef\xbb\xbf" + good, b" " * (16 << 10) + good,
              json.dumps({**json.loads(good), "kid": key_id(public_key())}).encode(),
              json.dumps({**json.loads(good), "kid": ["x"]}).encode(),
              json.dumps({**json.loads(good), "kid": "00000000"}).encode(),
              json.dumps({**json.loads(good), "signature": "!" + json.loads(good)["signature"]}).encode(),
              json.dumps({**json.loads(good), "signature": json.loads(good)["signature"][:-1]}).encode(),
              json.dumps({**json.loads(good), "signature": "é"}).encode(), b"\xff",
              answer({**RECORD, "page": "https://evil.example"}), answer([1])]
    cases = []
    for raw in bodies:
        result = outcome(lambda: service._verify(raw))
        if "ok" in result:
            result["ok"] = json.dumps(result["ok"], separators=(",", ":"))
        cases.append({"raw": base64.b64encode(raw).decode(), **result})
    return cases


def main():
    service = UpdateService(HERE, lambda: "https://license.scadabench.com", keys=keys())
    vectors = {
        "test_key": {"kid": key_id(public_key()), "public": public_key().hex()},
        "parse": [{"input": json.dumps(value), **outcome(lambda: versions.parse(value))} for value in VERSIONS],
        "newer": [{"a": json.dumps(a), "b": json.dumps(b), **outcome(lambda: versions.is_newer(a, b))}
                  for a in VERSIONS[:14] for b in ["0.1.13", "0.1.13+dev", "0.1.14", "garbage", None]],
        "safe_url": [{"input": json.dumps(url), "safe": safe(url)} for url in URLS],
        "records": [{"input": json.dumps(record, separators=(",", ":")),
                     **{k: (json.dumps(v, separators=(",", ":")) if k == "ok" else v)
                        for k, v in outcome(lambda: versions.clean_record(record)).items()}}
                    for record in records()],
        "verify": verify_cases(service),
        "scenarios": run_scenarios(),
    }
    (HERE / "generated").mkdir(exist_ok=True)
    (HERE / "generated" / "vectors.json").write_text(json.dumps(vectors, indent=0) + "\n", encoding="utf-8",
                                                     newline="\n")


if __name__ == "__main__":
    main()
