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

// install is LicenseService._install: verify, check it fits this PC, then store it.
func (s *Service) install(input any) (*pyjson.Object, error) {
	bundle, err := ParseBundle(input)
	if err != nil {
		return nil, err
	}
	payload, err := VerifyBundle(bundle, s.options.Keys)
	if err != nil {
		return nil, err
	}
	now := s.now()
	bound, err := s.machineFor(payload)
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
	if err := s.storeCertificate(bundle, now, s.trialStamp(input, payload)); err != nil {
		return nil, err
	}
	return s.Status()
}

// trialStamp is the signed server time a trial answer carries (a decoded server answer
// may have "server_time"); nil when there is none to remember.
func (s *Service) trialStamp(input any, payload *pyjson.Object) func() error {
	answer, isObject := input.(*pyjson.Object)
	if plan, _ := payload.Get("type"); plan != any("trial") || !isObject {
		return nil
	}
	stamp, _ := answer.Get("server_time")
	return func() error {
		_, err := s.trialClock.Remember(stamp)
		return err
	}
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

// Activate binds licenseKey (any decoded JSON value) to this PC; move transfers it.
func (s *Service) Activate(licenseKey any, move bool) (*pyjson.Object, error) {
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
	var bundle any
	if move {
		bundle, err = client.Transfer(key, hwid)
	} else {
		bundle, err = client.Activate(key, hwid)
	}
	if err != nil {
		return nil, err
	}
	return s.install(bundle)
}

// ActivateOffline installs a pasted license.cert (text or decoded object).
func (s *Service) ActivateOffline(certificate any) (*pyjson.Object, error) {
	if err := s.requireEnforced(); err != nil {
		return nil, err
	}
	status, err := s.install(certificate)
	var bad *CertificateError
	if errors.As(err, &bad) {
		return nil, &ActionError{Message: bad.Message, Code: "bad_certificate"}
	}
	return status, err
}

// SetServerURL stores the license server address in client_config.json (indent=2).
func (s *Service) SetServerURL(url any) (*pyjson.Object, error) {
	if err := s.requireEnforced(); err != nil {
		return nil, err
	}
	text := ""
	if truthy(url) {
		text, _ = scalarText(url)
		if text == "" {
			text = "[...]" // repr of a list or object: never a valid address
		}
	}
	text = strings.TrimRight(pytext.Strip(text), "/")
	if text != "" && !allowedServer(text) {
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
