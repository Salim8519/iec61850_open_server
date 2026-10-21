"""Own this computer's license state: certificate file, clock anchor and activation."""
import json
import os
from pathlib import Path
import re
import secrets
import threading
import time
from urllib.parse import urlparse
from studio_app.domain import license_policy, trial_policy
from . import certificate, integrity, release_key
from .client import LicenseClient
from .certificate import CertificateError
from .clock_store import ClockValue
from .online_check import OnlineChecks
from .revoked_ids import RevokedIds
from .time_anchor import TimeAnchor
from .trial_clock import TrialClock

KEY_PATTERN = re.compile(r"^SUBS(-[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{4}){4}$")
STATUS_CACHE_SECONDS = 60


class LicenseActionError(ValueError):
    def __init__(self, message, code="invalid"):
        super().__init__(message)
        self.code = code


def default_home():
    base = os.environ.get("LOCALAPPDATA") or str(Path.home() / "AppData/Local")
    return Path(base) / "ScadaBench"


def mask_key(license_id):
    if not license_id or not str(license_id).startswith("SUBS-"):
        return None
    return "SUBS-****-****-****-" + str(license_id)[-4:]


class LicenseService(OnlineChecks):
    def __init__(self, home, data_dir, enforce, hwid_provider=None, clock=time.time, keys=None,
                 client_factory=LicenseClient, native_folder=None, expected_files=None,
                 legacy_hwid_provider=None, registry=None, parts_provider=None,
                 allow_test_server=False, nonce_provider=None):
        self.home = Path(home)
        self.native_folder = native_folder
        self.expected_files = expected_files
        self.enforce = enforce
        self.clock = clock
        self.keys = keys
        self.client_factory = client_factory
        # Test-only seam: tests and the license sandbox talk to a server on this PC. The packaged
        # app never sets it, so an enforcing build only ever asks its built-in server (ADR-0097).
        self.allow_test_server = bool(allow_test_server)
        self._nonce_provider = nonce_provider or (lambda: secrets.token_hex(16))
        self.cert_file = self.home / "license.cert"
        self.config_file = self.home / "client_config.json"
        self.state_file = self.home / "license_state.json"
        # registry: clock_store.WindowsRegistry() in the app; tests pass a fake or None.
        self.anchor = TimeAnchor(ClockValue("time_anchor", self.home, registry), Path(data_dir) / "logs/runs.jsonl")
        self.trial_clock = TrialClock(self.home, registry, keys)
        self.revoked = RevokedIds(self.home, registry)
        self._parts_provider = parts_provider
        self._stop = threading.Event()
        self._hwid_provider = hwid_provider
        self._hwid = None
        self._legacy_hwid_provider = legacy_hwid_provider
        self._legacy_hwid = False
        self._bound_hwid = None
        self._lock = threading.RLock()
        self._cached = None
        self._cached_at = 0.0
        if enforce:
            os.environ["STUDIO_LICENSE_CERT_PATH"] = str(self.cert_file)

    # ----- state files -------------------------------------------------
    def _read_json(self, path, default):
        try:
            return json.loads(path.read_text(encoding="utf-8"))
        except (OSError, ValueError):
            return default

    def _write(self, path, text):
        self.home.mkdir(parents=True, exist_ok=True)
        temporary = path.with_name(path.name + ".tmp")
        temporary.write_text(text, encoding="utf-8")
        os.replace(temporary, path)

    def hwid(self):
        if self._hwid is None:
            from . import hwid as machine
            self._hwid = (self._hwid_provider or machine.machine_id)()
        return self._hwid

    def legacy_hwid(self):
        """This PC's pre-v2 machine ID, or None when it cannot be read or is not wanted."""
        if self._legacy_hwid is False:
            from . import hwid as machine
            provider = self._legacy_hwid_provider
            if provider is None and self._hwid_provider is None:
                provider = machine.legacy_machine_id
            try:
                self._legacy_hwid = provider() if provider else None
            except OSError:
                self._legacy_hwid = None
        return self._legacy_hwid

    def _machine_for(self, payload):
        """The machine ID this certificate must match (v1 only for pre-cutoff certificates)."""
        from . import hwid as machine
        try:
            return machine.bound_hwid(payload.get("hwid"), payload.get("issued_at", 0),
                                      self.hwid(), self.legacy_hwid())
        except (TypeError, ValueError, AttributeError):
            return self.hwid()

    def server_url(self):
        """The built-in server; client_config.json is read only in development and in tests."""
        if self.enforce and not self.allow_test_server:
            return (release_key.DEFAULT_SERVER_URL or "").rstrip("/")
        configured = self._read_json(self.config_file, {}).get("server_url")
        return (configured or release_key.DEFAULT_SERVER_URL or "").rstrip("/")

    def server_host(self):
        return urlparse(self.server_url()).hostname or "the license server"

    def trial_parts(self):
        """Salted part hashes for the server's trial check; {} when they cannot be read."""
        from . import trial_parts
        try:
            return (self._parts_provider or trial_parts.machine_parts)()
        except (OSError, ValueError):
            return {}

    # ----- evaluation --------------------------------------------------
    def _evaluate(self):
        if not self.enforce:
            return license_policy.development_status()
        tampered = integrity.problem(self.native_folder or ".", self.expected_files)
        if tampered:
            return license_policy.unlicensed(tampered)
        self.revoked.restore()  # a deleted copy of the cancelled IDs comes back from the others
        revoked = self._read_json(self.state_file, {}).get("revoked_reason")
        try:
            text = self.cert_file.read_text(encoding="utf-8")
        except OSError:
            return license_policy.unlicensed(revoked or license_policy.NO_LICENSE)
        try:
            payload = certificate.verify_bundle(certificate.parse_bundle(text), self.keys)
        except CertificateError as error:
            return license_policy.unlicensed(str(error))
        if self.revoked.contains(payload.get("license_id")):  # cancelled stays cancelled, whatever the file says
            return license_policy.unlicensed(license_policy.REVOKED, payload.get("type"), payload.get("license_id"))
        now = int(self.clock())
        self._bound_hwid = self._machine_for(payload)
        if payload.get("type") != "trial":  # lifetime: exactly the original rule
            status = license_policy.evaluate_certificate(payload, self._bound_hwid, now, self.anchor.max_seen(), False)
        else:
            status = self._evaluate_trial(payload, now)
        if status.licensed:
            self.anchor.advance(now)
        return status

    def _evaluate_trial(self, payload, now):
        confirmed = self.trial_clock.confirmed()
        if trial_policy.anchor_missing(self.anchor.value.read_ints(), confirmed):
            self.trial_clock.mark_suspicious(now)  # asks one quiet online check, never blocks now
        status = license_policy.evaluate_certificate(
            payload, self._bound_hwid, now, trial_policy.latest_seen(self.anchor.max_seen(), confirmed), False,
            trial_policy.TRIAL_CLOCK_TOLERANCE_SECONDS)
        return trial_policy.apply(status, confirmed, self.trial_clock.suspicious_since(), now, self.server_host())

    def current(self, refresh=False):
        with self._lock:
            age = self.clock() - self._cached_at
            if refresh or self._cached is None or not 0 <= age < STATUS_CACHE_SECONDS:
                self._cached, self._cached_at = self._evaluate(), self.clock()
            return self._cached

    def is_licensed(self):
        return self.current().licensed

    def status(self):
        current = self.current(refresh=True)
        result = current.as_dict()
        result.update(enforced=self.enforce, server_url=self.server_url() if self.enforce else "",
                      masked_key=mask_key(current.license_id),
                      hwid=self.hwid() if self.enforce else None,
                      transfer_available=False)  # kept for old screens; licenses never move (ADR-0097)
        return result

    # ----- actions -----------------------------------------------------
    def _require_enforced(self):
        if not self.enforce:
            raise LicenseActionError("Development mode does not use licenses.", "development")

    def _install(self, bundle):
        """Install a certificate (bare, or a server answer that may carry a signed time)."""
        payload = certificate.verify_bundle(certificate.parse_bundle(bundle), self.keys)
        if self.revoked.contains(payload.get("license_id")):
            raise LicenseActionError(license_policy.REVOKED, "not_valid_here")
        now = int(self.clock())
        status = license_policy.evaluate_certificate(payload, self._machine_for(payload), now,
                                                     self.anchor.max_seen(), False)
        if not status.licensed:
            raise LicenseActionError(status.reason or "This license cannot be used here.", "not_valid_here")
        with self._lock:
            self._write(self.cert_file, json.dumps(certificate.parse_bundle(bundle), indent=2))
            state = self._read_json(self.state_file, {})
            state.update(revoked_reason=None, last_verify=now)
            self._write(self.state_file, json.dumps(state))
            self.anchor.advance(now)
            if payload.get("type") == "trial" and isinstance(bundle, dict):
                self.trial_clock.remember(bundle.get("server_time"))
            self._cached = None
        return self.status()

    def _client(self):
        url = self.server_url()
        if self.client_factory is LicenseClient:
            return LicenseClient(url, test_server=self.allow_test_server)
        return self.client_factory(url)

    def start_trial(self, email=None, code=None):
        """Start (or get back) this PC's trial; email/code only when the server asked for them."""
        self._require_enforced()
        return self._install(self._client().start_trial(self.hwid(), self.trial_parts(), email, code))

    def request_trial_code(self, email):
        """Ask the server to email a trial code; "no_code_needed" when mail is unavailable."""
        self._require_enforced()
        address = str(email or "").strip()
        if "@" not in address or len(address) > 254:
            raise LicenseActionError(trial_policy.BAD_EMAIL, "bad_email")
        return {"status": self._client().request_trial_code(self.hwid(), address).get("status", "sent")}

    @staticmethod
    def normalize_key(license_key):
        key = re.sub(r"\s+", "", str(license_key or "")).upper()
        if not KEY_PATTERN.match(key):
            raise LicenseActionError("A license key looks like SUBS-XXXX-XXXX-XXXX-XXXX. "
                                     "Copy it from your purchase email.", "bad_key")
        return key

    def activate(self, license_key):
        self._require_enforced()
        key = self.normalize_key(license_key)
        return self._install(self._client().activate(key, self.hwid()))

    def activate_offline(self, certificate_text):
        self._require_enforced()
        try:
            return self._install(certificate_text)
        except CertificateError as error:
            raise LicenseActionError(str(error), "bad_certificate") from error

    def set_server_url(self, url):
        self._require_enforced()
        if not self.allow_test_server:
            raise LicenseActionError(license_policy.SERVER_FIXED, "bad_url")
        url = str(url or "").strip().rstrip("/")
        if url and not url.startswith(("https://", "http://127.0.0.1", "http://localhost")):
            raise LicenseActionError("The server address must start with https://", "bad_url")
        config = self._read_json(self.config_file, {})
        config["server_url"] = url
        self._write(self.config_file, json.dumps(config, indent=2))
        return self.status()
