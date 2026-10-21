"""Quiet daily update check (ADR-0071, change 0105): signed, private, never pushy."""
import base64
import io
import json
import re
from pathlib import Path
from urllib.error import HTTPError, URLError
from studio_app.updates import versions
from studio_app.updates.service import RELEASE_DOMAIN, UpdateService
from studio_app.version import __version__
from license_signer import keys, sign

ROOT = Path(__file__).resolve().parents[3]
RECORD = {"version": "0.1.14", "date": "2026-10-20", "notes": "Faster SLD.", "page": "https://scadabench.com/download",
          "installer": {"url": "https://scadabench.com/download/ScadaBench-Setup-0.1.14.exe", "sha256": "a" * 64}}


def answer(record=RECORD, domain=RELEASE_DOMAIN):
    payload = json.dumps(record, separators=(",", ":"))
    signature = base64.urlsafe_b64encode(sign(domain + payload.encode())).decode().rstrip("=")
    return json.dumps({"payload": payload, "signature": signature}).encode()


class Response(io.BytesIO):
    def __init__(self, body, etag='"e1"'):
        super().__init__(body)
        self.headers = {"ETag": etag}


class Opener:
    def __init__(self, *results):
        self.results, self.requests = list(results), []

    def __call__(self, request, timeout):
        self.requests.append(request)
        result = self.results.pop(0)
        if isinstance(result, Exception):
            raise result
        return Response(result)


def service(tmp_path, opener, current="0.1.13", clock=lambda: 1_000_000):
    return UpdateService(tmp_path, lambda: "https://license.scadabench.com", current=current,
                         keys=keys(), opener=opener, clock=clock)


def test_a_signed_newer_release_is_offered_and_nothing_about_the_pc_is_sent(tmp_path):
    opener = Opener(answer())
    status = service(tmp_path, opener).check()
    assert status["available"] and status["latest"]["version"] == "0.1.14" and not status["important"]
    request = opener.requests[0]
    assert request.full_url == "https://license.scadabench.com/api/v1/release" and request.data is None
    assert set(name.lower() for name in request.headers) <= {"user-agent", "accept", "if-none-match"}


def test_a_license_style_signature_or_a_foreign_link_is_refused(tmp_path):
    assert service(tmp_path, Opener(answer(domain=b""))).check()["error"] == "bad_answer"
    evil = {**RECORD, "installer": {"url": "https://evil.example/x.exe", "sha256": "a" * 64}}
    assert service(tmp_path, Opener(answer(evil))).check()["error"] == "bad_answer"
    assert service(tmp_path, Opener(answer(evil))).status()["latest"] is None


def test_offline_is_quiet_and_keeps_the_last_good_answer(tmp_path):
    first = service(tmp_path, Opener(answer()))
    first.check()
    second = service(tmp_path, Opener(URLError("no route")), clock=lambda: 2_000_000)
    status = second.check()
    assert status["error"] == "offline" and status["latest"]["version"] == "0.1.14"
    unchanged = service(tmp_path, Opener(HTTPError("u", 304, "same", {}, None)), clock=lambda: 3_000_000)
    assert unchanged.check()["error"] is None and opener_etag(unchanged) == '"e1"'


def opener_etag(update):
    return update.opener.requests[0].headers.get("If-none-match")


def test_once_a_day_dismiss_per_version_and_auto_off(tmp_path):
    update = service(tmp_path, Opener(answer()))
    assert update.due()
    update.check()
    assert not update.due(), "checked less than a day ago"
    assert update.check(manual=True)["latest"]["version"] == "0.1.14", "a repeated Check now reuses the answer"
    assert update.configure({"dismissed": "0.1.14"})["dismissed"] is True
    assert update.configure({"auto": False})["auto"] is False
    late = service(tmp_path, Opener(), clock=lambda: 1_000_000 + 2 * 86400)
    assert not late.due(), "automatic checks off"


def test_an_important_release_cannot_be_dismissed(tmp_path):
    update = service(tmp_path, Opener(answer({**RECORD, "min_version": "0.1.14"})))
    update.check()
    status = update.configure({"dismissed": "0.1.14"})
    assert status["important"] and status["dismissed"] is False


def test_versions_and_the_development_version_match_the_website():
    assert versions.is_newer("0.1.14", "0.1.13+dev") and not versions.is_newer("0.1.13", "0.1.13+dev")
    assert not versions.is_newer("garbage", "0.1.13") and versions.parse("0.10.2") == (0, 10, 2)
    site = (ROOT / "website/src/data/site.ts").read_text(encoding="utf-8")
    assert __version__.split("+")[0] == re.search(r"const VERSION = '([^']+)'", site).group(1), \
        "bump studio_app/version.py together with website/src/data/site.ts"
