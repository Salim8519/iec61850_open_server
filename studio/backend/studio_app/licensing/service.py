"""Own this computer's license state: certificate file, clock anchor and activation."""
import json
import logging
import os
from pathlib import Path
import re
import threading
import time
from studio_app.domain import license_policy
from . import certificate, integrity, release_key
from .client import LicenseClient, LicenseServerError
from .certificate import CertificateError
from .time_anchor import TimeAnchor

KEY_PATTERN = re.compile(r"^SUBS(-[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{4}){4}$")
STATUS_CACHE_SECONDS = 60
VERIFY_EVERY_SECONDS = 7 * 86400
logger = logging.getLogger(__name__)


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


class LicenseService:
    def __init__(self, home, data_dir, enforce, hwid_provider=None, clock=time.time, keys=None,
                 client_factory=LicenseClient, native_folder=None, expected_files=None):
        self.home = Path(home)
        self.native_folder = native_folder
        self.expected_files = expected_files
        self.enforce = enforce
        self.clock = clock
        self.keys = keys
        self.client_factory = client_factory
        self.cert_file = self.home / "license.cert"
        self.config_file = self.home / "client_config.json"
        self.state_file = self.home / "license_state.json"
        self.anchor = TimeAnchor(self.home / ".time_anchor", Path(data_dir) / "logs/runs.jsonl")
        self._hwid_provider = hwid_provider
        self._hwid = None
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

    def server_url(self):
        configured = self._read_json(self.config_file, {}).get("server_url")
        return (configured or release_key.DEFAULT_SERVER_URL or "").rstrip("/")

    # ----- evaluation --------------------------------------------------
    def _evaluate(self):
        if not self.enforce:
            return license_policy.development_status()
        tampered = integrity.problem(self.native_folder or ".", self.expected_files)
        if tampered:
            return license_policy.unlicensed(tampered)
        revoked = self._read_json(self.state_file, {}).get("revoked_reason")
        try:
            text = self.cert_file.read_text(encoding="utf-8")
        except OSError:
            return license_policy.unlicensed(revoked or license_policy.NO_LICENSE)
        try:
            payload = certificate.verify_bundle(certificate.parse_bundle(text), self.keys)
        except CertificateError as error:
            return license_policy.unlicensed(str(error))
        now = int(self.clock())
        status = license_policy.evaluate_certificate(payload, self.hwid(), now, self.anchor.max_seen(), False)
        if status.licensed:
            self.anchor.advance(now)
        return status

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
                      transfer_available=current.reason == license_policy.WRONG_MACHINE)
        return result

    # ----- actions -----------------------------------------------------
    def _require_enforced(self):
        if not self.enforce:
            raise LicenseActionError("Development mode does not use licenses.", "development")

    def _install(self, bundle):
        payload = certificate.verify_bundle(certificate.parse_bundle(bundle), self.keys)
        now = int(self.clock())
        status = license_policy.evaluate_certificate(payload, self.hwid(), now, self.anchor.max_seen(), False)
        if not status.licensed:
            raise LicenseActionError(status.reason or "This license cannot be used here.", "not_valid_here")
        with self._lock:
            self._write(self.cert_file, json.dumps(certificate.parse_bundle(bundle), indent=2))
            state = self._read_json(self.state_file, {})
            state.update(revoked_reason=None, last_verify=now)
            self._write(self.state_file, json.dumps(state))
            self.anchor.advance(now)
            self._cached = None
        return self.status()

    def _client(self):
        return self.client_factory(self.server_url())

    def start_trial(self):
        self._require_enforced()
        return self._install(self._client().start_trial(self.hwid()))

    @staticmethod
    def normalize_key(license_key):
        key = re.sub(r"\s+", "", str(license_key or "")).upper()
        if not KEY_PATTERN.match(key):
            raise LicenseActionError("A license key looks like SUBS-XXXX-XXXX-XXXX-XXXX. "
                                     "Copy it from your purchase email.", "bad_key")
        return key

    def activate(self, license_key, move=False):
        self._require_enforced()
        key = self.normalize_key(license_key)
        client = self._client()
        bundle = client.transfer(key, self.hwid()) if move else client.activate(key, self.hwid())
        return self._install(bundle)

    def activate_offline(self, certificate_text):
        self._require_enforced()
        try:
            return self._install(certificate_text)
        except CertificateError as error:
            raise LicenseActionError(str(error), "bad_certificate") from error

    def set_server_url(self, url):
        self._require_enforced()
        url = str(url or "").strip().rstrip("/")
        if url and not url.startswith(("https://", "http://127.0.0.1", "http://localhost")):
            raise LicenseActionError("The server address must start with https://", "bad_url")
        config = self._read_json(self.config_file, {})
        config["server_url"] = url
        self._write(self.config_file, json.dumps(config, indent=2))
        return self.status()

    # ----- background revocation check ---------------------------------
    def verify_online_if_due(self):
        """Learn about refunds or key moves; offline PCs simply keep their license."""
        if not self.enforce:
            return None
        status = self.current(refresh=True)
        state = self._read_json(self.state_file, {})
        if not status.licensed or status.plan != "lifetime" or \
                self.clock() - state.get("last_verify", 0) < VERIFY_EVERY_SECONDS:
            return None
        try:
            answer = self._client().verify(status.license_id, self.hwid())
        except LicenseServerError:
            return None
        verdict = answer.get("status")
        state["last_verify"] = int(self.clock())
        if verdict in {"revoked", "moved"}:
            state["revoked_reason"] = ("This license was cancelled. Contact the seller if this is a mistake."
                                       if verdict == "revoked" else
                                       "This license key was moved to another computer.")
            with self._lock:
                stamp = int(self.clock())
                os.replace(self.cert_file, self.cert_file.with_name(f"license.cert.{verdict}-{stamp}"))
                self._cached = None
        elif verdict == "valid" and answer.get("certificate"):
            try:
                fresh = certificate.parse_bundle(answer["certificate"])
                if certificate.verify_bundle(fresh, self.keys).get("hwid") == self.hwid():
                    self._write(self.cert_file, json.dumps(fresh, indent=2))
            except CertificateError:
                pass
        self._write(self.state_file, json.dumps(state))
        return verdict

    def start_background_verify(self):
        if self.enforce:
            threading.Thread(target=self._safe_verify, name="license-verify", daemon=True).start()

    def _safe_verify(self):
        try:
            self.verify_online_if_due()
        except Exception:
            logger.exception("Background license check failed")
