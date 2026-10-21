package licensing

import (
	"errors"
	"regexp"
	"strings"

	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/pyjson"
)

// KeyPattern is service.KEY_PATTERN (no I, L, O, 0 or 1 in a key).
var KeyPattern = regexp.MustCompile(`^SUBS(-[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{4}){4}$`)

// Action texts of licensing/service.py.
const (
	DevelopmentText  = "Development mode does not use licenses."
	BadKeyText       = "A license key looks like SUBS-XXXX-XXXX-XXXX-XXXX. Copy it from your purchase email."
	BadURLText       = "The server address must start with https://"
	NotValidHereText = "This license cannot be used here."
)

// upperSpecial are the full case mappings of str.upper() that turn one character into
// ASCII letters a key can contain; strings.ToUpper only applies simple mappings.
var upperSpecial = strings.NewReplacer("ß", "SS", "ﬀ", "FF", "ﬅ", "ST", "ﬆ", "ST")

// MaskKey is service.mask_key: "SUBS-****-****-****-" + the last four characters for a
// SUBS key, otherwise nil.
func MaskKey(licenseID any) any {
	text, isText := licenseID.(string)
	if !truthy(licenseID) || !isText || !strings.HasPrefix(text, "SUBS-") {
		return nil
	}
	runes := []rune(text)
	return "SUBS-****-****-****-" + string(runes[max(0, len(runes)-4):])
}

// NormalizeKey is LicenseService.normalize_key for any decoded JSON value.
func NormalizeKey(licenseKey any) (string, error) {
	text, _ := licenseKey.(string)
	key := strings.ToUpper(upperSpecial.Replace(pytext.RemoveSpace(text)))
	if !KeyPattern.MatchString(key) {
		return "", &ActionError{Message: BadKeyText, Code: "bad_key"}
	}
	return key, nil
}

func (s *Service) requireEnforced() error {
	if !s.options.Enforce {
		return &ActionError{Message: DevelopmentText, Code: "development"}
	}
	return nil
}

// install is LicenseService._install: verify, check it fits this PC, then store it. chip is
// the new attestation key of a chip-bound activation (nil otherwise); install keeps its blob
// as ak.blob and marks it stored.
func (s *Service) install(input any, chip *chipEnrollment) (*pyjson.Object, error) {
	bundle, err := ParseBundle(input)
	if err != nil {
		return nil, err
	}
	payload, err := VerifyBundle(bundle, s.options.Keys)
	if err != nil {
		return nil, err
	}
	if licenseID, _ := payload.Get("license_id"); s.revoked.Contains(licenseID) {
		return nil, &ActionError{Message: RevokedText, Code: "not_valid_here"}
	}
	now := s.now()
	bound, err := s.HWID()
	if err != nil {
		return nil, err
	}
	status, err := EvaluateCertificate(payload, bound, now, s.anchor.MaxSeen(), false, ClockToleranceSeconds)
	if err != nil {
		return nil, uncaught(err.Error())
	}
	if !status.Licensed {
		reason, _ := status.Reason.(string)
		if reason == "" {
			reason = NotValidHereText
		}
		return nil, &ActionError{Message: reason, Code: "not_valid_here"}
	}
	var previousKey []byte
	if chip != nil && certAK(payload) != "" {
		if previousKey, err = s.storeChipKey(chip); err != nil {
			return nil, err
		}
		chip.stored = true
	}
	if err := s.storeCertificate(bundle, now, s.afterInstall(input, payload)); err != nil {
		if chip != nil && chip.stored {
			s.restoreChipKey(previousKey)
		}
		return nil, err
	}
	if len(previousKey) > 0 {
		s.options.Chip.Forget(previousKey) // the key of the license this one replaces
	}
	return s.Status()
}

// afterInstall is what is remembered once the certificate is stored: a trial's signed server
// time, and a lease license's token (a new activation starts a new lease, so the old signed
// answer goes). It returns nil when there is nothing to remember.
func (s *Service) afterInstall(input any, payload *pyjson.Object) func() error {
	answer, isObject := input.(*pyjson.Object)
	if !isObject {
		return nil
	}
	plan, _ := payload.Get("type")
	stamp, _ := answer.Get("server_time")
	switch {
	case plan == any("trial"):
		return func() error {
			_, err := s.trialClock.Remember(stamp)
			return err
		}
	case leaseDays(payload) > 0 && certAK(payload) == "":
		token, _ := lookupString(answer, "lease_token")
		return func() error {
			s.lease.ClearReceipt()
			if ValidLeaseToken(token) {
				at, _, _ := VerifyServerTime(stamp, s.options.Keys)
				if at == 0 {
					at = s.now()
				}
				s.lease.SetToken(at, token)
			}
			return nil
		}
	}
	return nil
}

// storeCertificate writes license.cert (indent=2), clears the revoked reason, records
// the verify time, advances the anchor and remembers a trial's signed time (remember).
func (s *Service) storeCertificate(bundle Bundle, now int64, remember func() error) error {
	s.filesMu.Lock()
	defer s.filesMu.Unlock()
	text, err := pytext.DumpsIndent(bundle.Object())
	if err != nil {
		return err
	}
	if err := pytext.WriteReplace(s.certFile, text); err != nil {
		return err
	}
	state, err := readObject(s.stateFile)
	if err != nil {
		return err
	}
	state.Set("revoked_reason", nil).Set("last_verify", now)
	if err := s.writeState(state); err != nil {
		return err
	}
	s.anchor.Advance(now)
	if remember != nil {
		if err := remember(); err != nil {
			return err
		}
	}
	s.invalidate()
	return nil
}

func (s *Service) writeState(state *pyjson.Object) error {
	text, err := pyjson.Dumps(state)
	if err != nil {
		return err
	}
	return pytext.WriteReplace(s.stateFile, text)
}

func (s *Service) client() (Client, error) {
	url, err := s.ServerURL()
	if err != nil {
		return nil, err
	}
	return s.options.ClientFactory(url), nil
}

// Activate binds licenseKey (any decoded JSON value) to this PC. Licenses never move to
// another computer, so there is no transfer (ADR-0097).
func (s *Service) Activate(licenseKey any) (*pyjson.Object, error) {
	if err := s.requireEnforced(); err != nil {
		return nil, err
	}
	key, err := NormalizeKey(licenseKey)
	if err != nil {
		return nil, err
	}
	client, err := s.client()
	if err != nil {
		return nil, err
	}
	hwid, err := s.HWID()
	if err != nil {
		return nil, err
	}
	bundle, chip, err := s.activateOnline(client, key, hwid)
	if err != nil {
		return nil, err
	}
	status, err := s.install(bundle, chip)
	if chip != nil {
		if err != nil || !chip.stored {
			chip.enrolled.Discard() // the key is not used: do not leave it in the chip's store
		} else {
			chip.enrolled.Close()
		}
	}
	return status, err
}

// ActivateOffline installs a pasted license.cert (text or decoded object).
func (s *Service) ActivateOffline(certificate any) (*pyjson.Object, error) {
	if err := s.requireEnforced(); err != nil {
		return nil, err
	}
	status, err := s.install(certificate, nil)
	var bad *CertificateError
	if errors.As(err, &bad) {
		return nil, &ActionError{Message: bad.Message, Code: "bad_certificate"}
	}
	return status, err
}

// SetServerURL stores the license server address in client_config.json (indent=2), but only
// for the test-only seam: an enforcing build always uses its built-in server (ADR-0097).
func (s *Service) SetServerURL(url any) (*pyjson.Object, error) {
	if err := s.requireEnforced(); err != nil {
		return nil, err
	}
	if !s.options.AllowTestServer {
		return nil, &ActionError{Message: ServerFixedText, Code: "bad_url"}
	}
	text := ""
	if truthy(url) {
		text, _ = scalarText(url)
		if text == "" {
			text = "[...]" // repr of a list or object: never a valid address
		}
	}
	text = strings.TrimRight(pytext.Strip(text), "/")
	if text != "" && !allowedServer(text, true) {
		return nil, &ActionError{Message: BadURLText, Code: "bad_url"}
	}
	s.filesMu.Lock()
	config, err := readObject(s.configFile)
	if err == nil {
		var data []byte
		if data, err = pytext.DumpsIndent(config.Set("server_url", text)); err == nil {
			err = pytext.WriteReplace(s.configFile, data)
		}
	}
	s.filesMu.Unlock()
	if err != nil {
		return nil, err
	}
	return s.Status()
}
