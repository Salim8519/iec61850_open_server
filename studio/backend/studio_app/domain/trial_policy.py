"""Trial-only rules on top of license_policy (ADR-0046 addendum 2026-10-19), and their wording.

"Smart, not soft": only clear evidence stops a trial. A clock that goes backwards by
more than a generous tolerance against the latest time seen (any anchor copy or the
signed server time) is a rollback. A server that cannot be reached never stops a trial.
Only when every anchor copy and the signed time are gone (deleted) is one online check
asked for, within a 3-day grace. Keep the numbers equal to license_clock.cpp.
"""
from dataclasses import replace
from . import license_policy as base

TRIAL_CLOCK_TOLERANCE_SECONDS = 48 * 3600
TRIAL_CHECK_EVERY_SECONDS = 86400
MISSING_ANCHOR_GRACE_SECONDS = 3 * 86400
SUPPORT_URL = base.SUPPORT_URL

TRIAL_USED = ("This computer has already used its free trial. Buy a lifetime license to keep using "
              f"ScadaBench. If you think this is a mistake, contact support: {SUPPORT_URL}")
EMAIL_REQUIRED = ("To start the free trial, please confirm your email address. We ask this when this computer "
                  "or your network has started trials before, for example in a shared office. "
                  f"Help: {SUPPORT_URL}")
EMAIL_USED = ("This email address already had a free trial. Buy a lifetime license to keep using ScadaBench, "
              f"or contact support if you think this is a mistake: {SUPPORT_URL}")
EMAIL_DISPOSABLE = ("Throwaway email services cannot be used for the free trial. Please use your work or "
                    f"personal email address. Help: {SUPPORT_URL}")
BAD_EMAIL = "Please enter a valid email address, for example name@company.com."
BAD_CODE = ("That code is not right or has expired. Check the latest email from ScadaBench, or ask for a "
            f"new code. Help: {SUPPORT_URL}")
SERVER_ERRORS = {"trial_used": TRIAL_USED, "email_required": EMAIL_REQUIRED, "email_used": EMAIL_USED,
                 "email_disposable": EMAIL_DISPOSABLE, "bad_email": BAD_EMAIL, "bad_code": BAD_CODE}


def needs_online(host):
    return ("ScadaBench's trial records on this computer were removed, so the date has to be confirmed with "
            "the license server once. Connect to the internet and click \"Check online now\". On a company "
            f"network, ask your IT team to allow {host} (HTTPS, port 443). Paid licenses work offline. "
            f"Help: {SUPPORT_URL}")


def latest_seen(max_seen, confirmed_at):
    """The rollback reference for a trial: anchors, run log and the signed server time."""
    return max(max_seen or 0, confirmed_at or 0)


def anchor_missing(anchor_copies, confirmed_at):
    """Every anchor copy and the signed time are gone: suspicious, never a block by itself."""
    return not anchor_copies and not confirmed_at


def check_due(confirmed_at, suspicious_since, now_ts):
    """A quiet online check is due daily, and at once after the anchor went missing."""
    return not confirmed_at or suspicious_since > confirmed_at or now_ts >= confirmed_at + TRIAL_CHECK_EVERY_SECONDS


def apply(status, confirmed_at, suspicious_since, now_ts, host):
    """Trial-only rules; other plans and unlicensed states pass through unchanged."""
    if not status.licensed or status.plan != "trial":
        return status
    if confirmed_at and status.expires_at and confirmed_at >= status.expires_at:
        return base.unlicensed(base.TRIAL_ENDED, "trial", status.license_id, status.expires_at)
    unconfirmed = suspicious_since and confirmed_at < suspicious_since
    if unconfirmed and now_ts >= suspicious_since + MISSING_ANCHOR_GRACE_SECONDS:
        blocked = base.unlicensed(needs_online(host), "trial", status.license_id, status.expires_at)
        return replace(blocked, online_check_required=True)
    return status
