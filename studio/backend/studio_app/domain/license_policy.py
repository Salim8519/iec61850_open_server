"""Decide whether a verified license payload entitles this computer to run ScadaBench.

Pure rules only: the caller has already checked the Ed25519 signature and gathered
the machine ID, the current time and the latest time Studio has observed.
"""
from dataclasses import asdict, dataclass
import math

CLOCK_TOLERANCE_SECONDS = 300
SUPPORT_URL = "https://scadabench.com/contact/"

NO_LICENSE = "This computer has no license yet. Start the free trial or enter your license key."
WRONG_MACHINE = ("This license belongs to a different computer. Enter your license key to move it "
                 "here, or start a trial.")
TRIAL_ENDED = "Your free trial has ended. Buy a lifetime license to keep using ScadaBench."
CLOCK_ROLLBACK = ("System clock rollback detected: the date on this computer is earlier than when ScadaBench "
                  "last ran. Correct the date and time in Windows (Settings > Time & language > Set time "
                  f"automatically), then reopen ScadaBench. Help: {SUPPORT_URL}")
DAMAGED = "The license file is damaged or was changed. Activate again or load a fresh license file."
UNSUPPORTED = "This license file is from a newer ScadaBench version. Update ScadaBench, then try again."


@dataclass(frozen=True)
class LicenseStatus:
    licensed: bool
    mode: str
    plan: str | None = None
    days_remaining: int | None = None
    expires_at: int | None = None
    reason: str | None = None
    license_id: str | None = None
    email: str | None = None
    online_check_required: bool = False

    def as_dict(self):
        return asdict(self)


def development_status():
    return LicenseStatus(licensed=True, mode="development", plan="dev")


def unlicensed(reason, plan=None, license_id=None, expires_at=None):
    return LicenseStatus(licensed=False, mode="unlicensed", plan=plan, reason=reason,
                         license_id=license_id, expires_at=expires_at)


def evaluate_certificate(payload, current_hwid, now_ts, max_seen_ts, is_dev_mode,
                         tolerance=CLOCK_TOLERANCE_SECONDS):
    """Return the licence status for an already signature-verified payload dict.

    tolerance is how far the clock may sit behind the latest time seen; trials pass a
    generous one (trial_policy.TRIAL_CLOCK_TOLERANCE_SECONDS), lifetime keeps 300 s.
    """
    if is_dev_mode:
        return development_status()
    if payload is None:
        return unlicensed(NO_LICENSE)
    try:
        version = int(payload.get("version", 0))
        plan = str(payload["type"])
        hwid = str(payload["hwid"])
        issued_at = int(payload["issued_at"])
        expires_at = int(payload.get("expires_at", 0))
    except (KeyError, TypeError, ValueError):
        return unlicensed(DAMAGED)
    license_id = payload.get("license_id")
    if version != 1:
        return unlicensed(UNSUPPORTED)
    if plan not in {"trial", "lifetime"}:
        return unlicensed(DAMAGED)
    if hwid != current_hwid:
        return unlicensed(WRONG_MACHINE, plan, license_id)
    if now_ts + tolerance < max(max_seen_ts or 0, issued_at):
        return unlicensed(CLOCK_ROLLBACK, plan, license_id, expires_at or None)
    if expires_at > 0 and now_ts >= expires_at:
        return unlicensed(TRIAL_ENDED if plan == "trial" else "Your license has expired.",
                          plan, license_id, expires_at)
    days = math.ceil((expires_at - now_ts) / 86400) if expires_at > 0 else None
    return LicenseStatus(licensed=True, mode=plan, plan=plan, days_remaining=days,
                         expires_at=expires_at or None, license_id=license_id,
                         email=payload.get("email") or None)

