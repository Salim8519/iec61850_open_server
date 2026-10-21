package licensing

import "math"

// Trial-only rules on top of the license policy (domain/trial_policy.py, ADR-0046
// addendum 2026-10-19), and their wording. "Smart, not soft": only clear evidence stops a
// trial. A clock more than TrialClockToleranceSeconds behind the latest time seen (any
// anchor copy or the signed server time) is a rollback; a server that cannot be reached
// never stops a trial. Only when every anchor copy and the signed time are gone is one
// online check asked for, within MissingAnchorGraceSeconds. Keep equal to license_clock.cpp.
const (
	TrialClockToleranceSeconds = 48 * 3600
	TrialCheckEverySeconds     = 86400
	MissingAnchorGraceSeconds  = 3 * 86400
)

// Trial texts, byte-identical to trial_policy.py.
const (
	TrialUsedText = "This computer has already used its free trial. Buy a lifetime license to keep using " +
		"ScadaBench. If you think this is a mistake, contact support: " + SupportURL
	EmailRequiredText = "To start the free trial, please confirm your email address. We ask this when this computer " +
		"or your network has started trials before, for example in a shared office. Help: " + SupportURL
	EmailUsedText = "This email address already had a free trial. Buy a lifetime license to keep using ScadaBench, " +
		"or contact support if you think this is a mistake: " + SupportURL
	EmailDisposableText = "Throwaway email services cannot be used for the free trial. Please use your work or " +
		"personal email address. Help: " + SupportURL
	BadEmailText = "Please enter a valid email address, for example name@company.com."
	BadCodeText  = "That code is not right or has expired. Check the latest email from ScadaBench, or ask for a " +
		"new code. Help: " + SupportURL
)

// ServerErrors are trial_policy.SERVER_ERRORS: server refusals shown in the app's own words.
var ServerErrors = map[string]string{"trial_used": TrialUsedText, "email_required": EmailRequiredText,
	"email_used": EmailUsedText, "email_disposable": EmailDisposableText, "bad_email": BadEmailText,
	"bad_code": BadCodeText}

// NeedsOnlineText is trial_policy.needs_online(host).
func NeedsOnlineText(host string) string {
	return "ScadaBench's trial records on this computer were removed, so the date has to be confirmed with " +
		"the license server once. Connect to the internet and click \"Check online now\". On a company " +
		"network, ask your IT team to allow " + host + " (HTTPS, port 443). Paid licenses work offline. " +
		"Help: " + SupportURL
}

// LatestSeen is the rollback reference for a trial: anchors, run log and signed server time.
func LatestSeen(maxSeen, confirmedAt int64) int64 { return max(maxSeen, confirmedAt, 0) }

// AnchorMissing reports that every anchor copy and the signed time are gone: suspicious,
// never a block by itself.
func AnchorMissing(anchorCopies int, confirmedAt int64) bool {
	return anchorCopies == 0 && confirmedAt == 0
}

// CheckDue reports that a quiet online check is due: daily, and at once after the anchor
// went missing.
func CheckDue(confirmedAt, suspiciousSince, now int64) bool {
	return confirmedAt == 0 || suspiciousSince > confirmedAt || now >= addSeconds(confirmedAt, TrialCheckEverySeconds)
}

// ApplyTrial is trial_policy.apply: trial-only rules; other plans and unlicensed states
// pass through unchanged.
func ApplyTrial(status Status, confirmedAt, suspiciousSince, now int64, host string) Status {
	if !status.Licensed || status.Plan != "trial" {
		return status
	}
	expires, _ := status.ExpiresAt.(int64)
	if confirmedAt != 0 && expires != 0 && confirmedAt >= expires {
		return Unlicensed(TrialEnded, "trial", status.LicenseID, status.ExpiresAt)
	}
	unconfirmed := suspiciousSince != 0 && confirmedAt < suspiciousSince
	if unconfirmed && now >= addSeconds(suspiciousSince, MissingAnchorGraceSeconds) {
		blocked := Unlicensed(NeedsOnlineText(host), "trial", status.LicenseID, status.ExpiresAt)
		blocked.OnlineCheckRequired = true
		return blocked
	}
	return status
}

// addSeconds adds without wrapping: a saturated (absurd) time stays the largest one, as
// Python's unbounded int would compare.
func addSeconds(moment, seconds int64) int64 {
	if moment > math.MaxInt64-seconds {
		return math.MaxInt64
	}
	return moment + seconds
}
