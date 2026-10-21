package licensing

import "studio-engine/internal/pyjson"

// How a lifetime license is tied to its computer (ADR-0098, docs/architecture/
// license-binding-contract.md). The signed certificate decides:
//
//   - "ak" present: the license is tied to the computer's security chip (TPM). At every
//     evaluation the chip signs a fresh nonce with the attestation key stored in ak.blob and
//     the signature is checked offline with the certificate's "ak". No online check is ever
//     needed. (chip.go, activate_chip.go)
//   - "lease_days" present: no chip could be proven, so the license is a lease. The server
//     passes a one-time lease token on at every successful check; a copy on another
//     computer holds a stale token and is cancelled. The license works offline for
//     lease_days after the last valid signed answer. (lease.go, lease_check.go)
//   - neither: an older certificate (and every trial), checked exactly as before.
const (
	BindingTPM   = "tpm"
	BindingLease = "lease"
)

// Texts of the binding rules.
const (
	// ChipFailedText is shown when the chip cannot confirm a chip-bound license.
	ChipFailedText = "This license is tied to this computer's security chip, which could not confirm it. " +
		"Check that the security chip (TPM) is switched on, then restart ScadaBench. If the chip was " +
		"cleared, enter your license key again. Help: " + SupportURL
	// LeaseExpiredText is shown when a lease license was not confirmed online for lease_days.
	LeaseExpiredText = "Connect this computer to the internet once so ScadaBench can confirm the license. " +
		"Computers with a security chip (TPM) do not need this."
)

// maxLeaseDays keeps lease_days * 86400 far from overflow; a real lease is 30 days.
const maxLeaseDays = 3650

// certAK is the attestation key (base64 TPMT_PUBLIC) a chip-bound certificate carries, or "".
func certAK(payload *pyjson.Object) string {
	value, _ := payload.Get("ak")
	text, _ := value.(string)
	return text
}

// leaseDays is the certificate's lease length in days, 0 when it is not a lease license.
func leaseDays(payload *pyjson.Object) int64 {
	value, found := payload.Get("lease_days")
	if !found {
		return 0
	}
	days, err := pyInt(value)
	if err != nil || days <= 0 {
		return 0
	}
	return min(days, maxLeaseDays)
}

// applyBinding adds the binding rule to a licensed lifetime status.
func (s *Service) applyBinding(status Status, payload *pyjson.Object, hwid string, now int64) Status {
	if ak := certAK(payload); ak != "" {
		return s.applyChip(status, ak)
	}
	if days := leaseDays(payload); days > 0 {
		return s.applyLease(status, payload, hwid, days, now)
	}
	return status
}

// applyChip asks the chip to prove the license is on this computer.
func (s *Service) applyChip(status Status, ak string) Status {
	if err := s.proveChip(ak); err != nil {
		s.options.Logf("The security chip could not confirm the license: %v", err)
		blocked := Unlicensed(ChipFailedText, status.Plan, status.LicenseID, nil)
		blocked.Binding = BindingTPM
		return blocked
	}
	status.Binding = BindingTPM
	return status
}

// applyLease blocks a lease license that was not confirmed within its lease.
func (s *Service) applyLease(status Status, payload *pyjson.Object, hwid string, days, now int64) Status {
	issued, _, _ := payloadInt(payload, "issued_at", false)
	valid := max(issued, s.lease.ReceiptTime(payload, hwid))
	s.mu.Lock()
	s.leaseValidFrom = valid
	s.mu.Unlock()
	status.Binding = BindingLease
	if now <= addSeconds(valid, days*86400) {
		return status
	}
	blocked := Unlicensed(LeaseExpiredText, status.Plan, status.LicenseID, nil)
	blocked.OnlineCheckRequired = true
	blocked.Binding = BindingLease
	return blocked
}
