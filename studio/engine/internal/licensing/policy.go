package licensing

import (
	"math"

	"studio-engine/internal/pyjson"
)

// ClockToleranceSeconds is how far the clock may lag the newest time ScadaBench saw
// (lifetime licenses; trials pass TrialClockToleranceSeconds).
const ClockToleranceSeconds = 300

// SupportURL is license_policy.SUPPORT_URL, linked from every trial and clock message.
const SupportURL = "https://scadabench.com/contact/"

// User-visible reasons, byte-identical to domain/license_policy.py.
const (
	NoLicense      = "This computer has no license yet. Start the free trial or enter your license key."
	WrongMachine   = "This license belongs to a different computer. Each license works on one computer only; buy a new license for this computer, or start a trial."
	TrialEnded     = "Your free trial has ended. Buy a lifetime license to keep using ScadaBench."
	ClockRollback  = "System clock rollback detected: the date on this computer is earlier than when ScadaBench last ran. Correct the date and time in Windows (Settings > Time & language > Set time automatically), then reopen ScadaBench. Help: " + SupportURL
	Damaged        = "The license file is damaged or was changed. Activate again or load a fresh license file."
	Unsupported    = "This license file is from a newer ScadaBench version. Update ScadaBench, then try again."
	LicenseExpired = "Your license has expired."
)

// Status is license_policy.LicenseStatus. Optional fields hold nil (Python None) or the
// value Python stores: Plan and Reason strings, DaysRemaining and ExpiresAt int64, and
// LicenseID and Email exactly as decoded from the signed payload. OnlineCheckRequired is
// set only when a trial must confirm its date online (trial_policy.apply).
type Status struct {
	Licensed            bool
	Mode                string
	Plan                any
	DaysRemaining       any
	ExpiresAt           any
	Reason              any
	LicenseID           any
	Email               any
	OnlineCheckRequired bool
}

// Object returns LicenseStatus.as_dict() with its field order.
func (s Status) Object() *pyjson.Object {
	return pyjson.NewObject().Set("licensed", s.Licensed).Set("mode", s.Mode).Set("plan", s.Plan).
		Set("days_remaining", s.DaysRemaining).Set("expires_at", s.ExpiresAt).Set("reason", s.Reason).
		Set("license_id", s.LicenseID).Set("email", s.Email).Set("online_check_required", s.OnlineCheckRequired)
}

// DevelopmentStatus is what development builds always report.
func DevelopmentStatus() Status { return Status{Licensed: true, Mode: "development", Plan: "dev"} }

// Unlicensed is license_policy.unlicensed(reason, plan, license_id, expires_at).
func Unlicensed(reason, plan, licenseID, expiresAt any) Status {
	return Status{Mode: "unlicensed", Plan: plan, Reason: reason, LicenseID: licenseID, ExpiresAt: expiresAt}
}

func orNone(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}

// payloadInt is int(payload[key]) (or payload.get(key, 0) when optional); damaged is
// set for the KeyError/TypeError/ValueError Python turns into DAMAGED.
func payloadInt(payload *pyjson.Object, key string, optional bool) (value int64, damaged bool, err error) {
	raw, found := payload.Get(key)
	if !found {
		return 0, !optional, nil
	}
	value, err = pyInt(raw)
	if err == errNotInt {
		return 0, true, nil
	}
	return value, false, err
}

// EvaluateCertificate is license_policy.evaluate_certificate for an already
// signature-verified payload (nil payload means no certificate). tolerance is how far the
// clock may sit behind the latest time seen: ClockToleranceSeconds for lifetime licenses,
// TrialClockToleranceSeconds for trials. An error stands for the OverflowError Python does
// not catch (an infinite number in the payload).
func EvaluateCertificate(payload *pyjson.Object, currentHWID string, now, maxSeen int64, isDev bool,
	tolerance int64) (Status, error) {
	if isDev {
		return DevelopmentStatus(), nil
	}
	if payload == nil {
		return Unlicensed(NoLicense, nil, nil, nil), nil
	}
	version, damaged, err := payloadInt(payload, "version", true)
	if err != nil || damaged {
		return Unlicensed(Damaged, nil, nil, nil), err
	}
	rawType, hasType := payload.Get("type")
	rawHWID, hasHWID := payload.Get("hwid")
	if !hasType || !hasHWID {
		return Unlicensed(Damaged, nil, nil, nil), nil
	}
	issuedAt, damaged, err := payloadInt(payload, "issued_at", false)
	if err != nil || damaged {
		return Unlicensed(Damaged, nil, nil, nil), err
	}
	expiresAt, damaged, err := payloadInt(payload, "expires_at", true)
	if err != nil || damaged {
		return Unlicensed(Damaged, nil, nil, nil), err
	}
	licenseID, _ := payload.Get("license_id")
	if version != 1 {
		return Unlicensed(Unsupported, nil, nil, nil), nil
	}
	plan, _ := rawType.(string)
	if plan != "trial" && plan != "lifetime" {
		return Unlicensed(Damaged, nil, nil, nil), nil
	}
	if hwid, ok := scalarText(rawHWID); !ok || hwid != currentHWID {
		return Unlicensed(WrongMachine, plan, licenseID, nil), nil
	}
	if now+tolerance < max(maxSeen, issuedAt) {
		return Unlicensed(ClockRollback, plan, licenseID, orNone(expiresAt)), nil
	}
	if expiresAt > 0 && now >= expiresAt {
		reason := LicenseExpired
		if plan == "trial" {
			reason = TrialEnded
		}
		return Unlicensed(reason, plan, licenseID, expiresAt), nil
	}
	var days any
	if expiresAt > 0 {
		days = int64(math.Ceil(float64(expiresAt-now) / 86400))
	}
	email, _ := payload.Get("email")
	if !truthy(email) {
		email = nil
	}
	return Status{Licensed: true, Mode: plan, Plan: plan, DaysRemaining: days,
		ExpiresAt: orNone(expiresAt), LicenseID: licenseID, Email: email}, nil
}
