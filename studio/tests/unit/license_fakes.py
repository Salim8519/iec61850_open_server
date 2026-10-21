"""Test doubles for the trial clock: an in-memory registry and a scriptable license server."""
import json
from license_signer import HWID, bundle, keys, stamp, verify_answer
from studio_app.licensing.client import LicenseServerError
from studio_app.licensing.service import LicenseService

NOW = 1_760_000_000
DAY = 86400


class FakeRegistry:
    """Stands in for HKCU\\Software\\ScadaBench, so tests never touch the real registry."""

    def __init__(self):
        self.values = {}

    def get(self, name):
        return self.values.get(name)

    def set(self, name, text):
        self.values[name] = str(text)


class Server:
    """What the fake license server answers: its clock, and whether it can be reached."""

    def __init__(self, time=NOW, online=True, signs=True, needs_email=False, mail_down=False, verdict="valid",
                 signs_answers=True):
        self.time, self.online, self.signs = time, online, signs
        self.verdict, self.signs_answers = verdict, signs_answers  # what "verify" says, and whether it signs it
        self.needs_email, self.mail_down = needs_email, mail_down
        self.calls = []

    def client(self, url):
        return FakeClient(self)


class FakeClient:
    def __init__(self, server):
        self.server = server

    def _answer(self, call, **answer):
        self.server.calls.append(call)
        if not self.server.online:
            raise LicenseServerError("Cannot reach the license server.", "offline")
        if self.server.signs:
            answer["server_time"] = stamp(self.server.time)
        return answer

    def start_trial(self, machine, parts=None, email=None, code=None):
        if self.server.needs_email and not (email and (self.server.mail_down or code == "123456")):
            self.server.calls.append("trial")
            raise LicenseServerError("Please confirm your email address.", "email_required")
        certificate = json.loads(bundle(type="trial", issued_at=NOW, expires_at=NOW + 7 * DAY, hwid=machine,
                                        license_id="TRIAL-" + machine[:12]))
        return self._answer("trial", certificate=certificate)

    def request_trial_code(self, machine, email):
        return self._answer("email-code", status="no_code_needed" if self.server.mail_down else "sent")

    def activate(self, key, machine):
        return self._answer("activate", certificate=json.loads(bundle(license_id=key, hwid=machine)))

    def verify(self, key, machine, nonce):
        answer = {"status": self.server.verdict}  # the unsigned copy is never trusted
        if self.server.signs_answers:
            answer["answer"] = verify_answer(nonce, key, machine, self.server.verdict, self.server.time)
        return self._answer("verify", **answer)


def studio(tmp_path, at, server=None, registry=None, **options):
    """A packaged-mode service whose clock reads `at` (a one-item list, so tests can move it)."""
    server = server or Server(online=False)
    return LicenseService(tmp_path / "home", tmp_path / "data", True, hwid_provider=lambda: HWID,
                          clock=lambda: at[0], keys=keys(), client_factory=server.client,
                          registry=registry, parts_provider=lambda: {}, **options)


def install_trial(tmp_path):
    home = tmp_path / "home"
    home.mkdir(parents=True, exist_ok=True)
    (home / "license.cert").write_text(bundle(type="trial", issued_at=NOW, expires_at=NOW + 7 * DAY,
                                              license_id="TRIAL-" + HWID[:12]), encoding="utf-8")
