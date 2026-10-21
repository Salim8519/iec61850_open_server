package licensing

// LeaseCheckEverySeconds is how often a lease license quietly asks the server again once it
// was confirmed (the background loop looks every BackgroundCheckSeconds). Every valid answer
// rotates the lease token, so the check is kept rare enough that a lost answer is a
// remote chance, yet far inside the 30-day lease.
const LeaseCheckEverySeconds = 6 * 3600

// verifyLease confirms a lease license online (ADR-0098): the app sends its lease token, the
// server answers (signed) with the verdict and a new token. A valid answer is stored as the
// receipt that renews the lease; "moved" means the token was stale, i.e. this is a copy, and
// cancels the license on this computer; "revoked" cancels it too. Unless force is set the
// check runs only when the last confirmation is older than LeaseCheckEverySeconds or the
// lease has run out. Failures to reach the server are returned as *ServerError (the
// background loop ignores them, "Check online now" shows them). Checks never overlap: a
// second one would present a token the first already replaced.
func (s *Service) verifyLease(status Status, force bool) (any, error) {
	if status.Binding != BindingLease || !truthy(status.LicenseID) {
		return nil, nil
	}
	s.leaseMu.Lock()
	defer s.leaseMu.Unlock()
	s.mu.Lock()
	confirmed := s.leaseValidFrom
	s.mu.Unlock()
	now := s.now()
	age := now - confirmed
	if age < 0 {
		age = -age // a clock slightly behind the server's must not make every check due
	}
	if !force && status.Licensed && confirmed > 0 && age < LeaseCheckEverySeconds {
		return nil, nil
	}
	hwid, err := s.HWID()
	if err != nil {
		return nil, err
	}
	verified, answer, err := s.signedAnswer(status.LicenseID, hwid, s.lease.CurrentToken())
	if err != nil {
		return nil, err
	}
	if verified.Status == "valid" {
		s.lease.SetReceipt(verified.Bundle)
		if verified.LeaseToken != "" {
			s.lease.SetToken(verified.Time, verified.LeaseToken)
		}
	}
	return s.applyVerdict(status.LicenseID, verified.Status, answer)
}
