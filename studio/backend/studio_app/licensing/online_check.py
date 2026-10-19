"""Background license checks: weekly revocation check for lifetime keys (unchanged) and
a quiet daily signed-time confirmation for trials. Failures never count against the
user: an offline PC simply keeps its license, lifetime or trial.
"""
import json
import logging
import os
import threading
from studio_app.domain import trial_policy
from . import certificate
from .certificate import CertificateError
from .client import LicenseServerError

VERIFY_EVERY_SECONDS = 7 * 86400
BACKGROUND_CHECK_SECONDS = 30 * 60
logger = logging.getLogger(__name__)


class OnlineChecks:
    """Mixed into LicenseService; uses its clock, files, client and lock."""

    def verify_online_if_due(self, trial_only=False):
        """Learn about refunds or key moves, or confirm a trial's date; never raises offline."""
        if not self.enforce:
            return None
        status = self.current(refresh=True)
        if status.plan == "trial" and status.license_id and (status.licensed or status.online_check_required):
            if not trial_policy.check_due(self.trial_clock.confirmed(), self.trial_clock.suspicious_since(),
                                          int(self.clock())):
                return None
            try:
                return self._confirm_trial_time(status)
            except LicenseServerError:
                return None
        if trial_only:
            return None
        return self._verify_lifetime(status)

    def check_online_now(self):
        """The "Check online now" button: confirm the trial date immediately, or explain why not."""
        self._require_enforced()
        status = self.current(refresh=True)
        if status.plan == "trial" and status.license_id:
            self._confirm_trial_time(status)
        return self.status()

    def _confirm_trial_time(self, status):
        answer = self._client().verify(status.license_id, self._bound_hwid or self.hwid())
        if not self.trial_clock.remember(answer.get("server_time")):
            raise LicenseServerError("The license server answered without a valid signed time. Try again later. "
                                     f"Help: {trial_policy.SUPPORT_URL}", "bad_time")
        with self._lock:
            self._cached = None
        return "confirmed"

    def _verify_lifetime(self, status):
        state = self._read_json(self.state_file, {})
        if not status.licensed or status.plan != "lifetime" or \
                self.clock() - state.get("last_verify", 0) < VERIFY_EVERY_SECONDS:
            return None
        try:
            # Ask about the machine ID the key is bound to; a legacy v1 binding would
            # otherwise look "moved" to the server and cancel a valid license.
            bound = self._bound_hwid or self.hwid()
            answer = self._client().verify(status.license_id, bound)
        except LicenseServerError:
            return None
        verdict = answer.get("status")
        update = {"last_verify": int(self.clock())}
        if verdict in {"revoked", "moved"}:
            update["revoked_reason"] = ("This license was cancelled. Contact the seller if this is a mistake."
                                        if verdict == "revoked" else
                                        "This license key was moved to another computer.")
            with self._lock:
                stamp = int(self.clock())
                os.replace(self.cert_file, self.cert_file.with_name(f"license.cert.{verdict}-{stamp}"))
                self._cached = None
        elif verdict == "valid" and answer.get("certificate") and bound == self.hwid():
            # A legacy-bound certificate is kept as is: a reissued one would carry a new
            # issued_at and no longer qualify for the legacy machine ID.
            try:
                fresh = certificate.parse_bundle(answer["certificate"])
                if certificate.verify_bundle(fresh, self.keys).get("hwid") == self.hwid():
                    self._write(self.cert_file, json.dumps(fresh, indent=2))
            except CertificateError:
                pass
        with self._lock:
            state = self._read_json(self.state_file, {})
            state.update(update)
            self._write(self.state_file, json.dumps(state))
        return verdict

    def start_background_verify(self):
        if self.enforce:
            threading.Thread(target=self._verify_loop, name="license-verify", daemon=True).start()

    def _verify_loop(self):
        self._safe_verify(trial_only=False)  # lifetime keys: once per start, as before
        while not self._stop.wait(BACKGROUND_CHECK_SECONDS):
            self._safe_verify(trial_only=True)

    def _safe_verify(self, trial_only):
        try:
            self.verify_online_if_due(trial_only=trial_only)
        except Exception:
            logger.exception("Background license check failed")
