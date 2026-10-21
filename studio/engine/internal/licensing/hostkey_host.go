package licensing

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"studio-engine/internal/licensing/hostkey"
	"studio-engine/internal/licensing/pytext"
)

// HostOptions are the seams of the licensed PC that answers its virtual machines (the host
// helper, cmd/scadabench_hostkey). Zero values mean this PC's real ones.
type HostOptions struct {
	Home      string // license home (license.cert, host_key.dpapi, the cancelled-ID record)
	Keys      map[string][]byte
	HWID      func() (string, error) // nil means MachineID
	Protector hostkey.Protector
	Registry  Registry
	Clock     func() float64
}

// HostResponder answers proof requests (hostkey.Responder). Before every answer it checks, from
// the files, that this PC may serve at all: its machine ID is the certificate's, the certificate
// is a genuine lifetime license that is not cancelled, and the DPAPI key opens and matches the
// certificate's host key. Anything else is a refusal in plain words.
type HostResponder struct{ options HostOptions }

// NewHostResponder returns the responder of the license in options.Home.
func NewHostResponder(options HostOptions) *HostResponder {
	if options.HWID == nil {
		options.HWID = MachineID
	}
	if options.Clock == nil {
		options.Clock = func() float64 { return float64(time.Now().UnixNano()) / 1e9 }
	}
	return &HostResponder{options: options}
}

// hostLicense is a license this PC may share: its certificate, public key and private key.
type hostLicense struct {
	bundle  Bundle
	public  string
	private ed25519.PrivateKey
}

// Check applies every rule of the host and returns the license it may share, or the refusal.
func (h *HostResponder) Check() (*hostLicense, error) {
	o := h.options
	text, err := pytext.ReadText(os.ReadFile(filepath.Join(o.Home, "license.cert")))
	if err != nil {
		return nil, errors.New(HostNoCertificateText)
	}
	bundle, err := ParseBundle(text)
	if err != nil {
		return nil, errors.New(HostNoCertificateText)
	}
	payload, err := VerifyBundle(bundle, o.Keys)
	if err != nil {
		return nil, err
	}
	if plan, _ := payload.Get("type"); plan != any("lifetime") {
		return nil, errors.New(HostNotLifetimeText)
	}
	hwid, err := o.HWID()
	if err != nil {
		return nil, err
	}
	status, err := EvaluateCertificate(payload, hwid, int64(o.Clock()), 0, false, ClockToleranceSeconds)
	if err != nil {
		return nil, err
	}
	if !status.Licensed {
		reason, _ := status.Reason.(string)
		return nil, errors.New(reason)
	}
	if licenseID, _ := payload.Get("license_id"); NewRevokedIDs(o.Home, o.Registry).Contains(licenseID) {
		return nil, errors.New(RevokedText)
	}
	public := certHostKey(payload)
	if public == "" {
		return nil, errors.New(HostNoKeyInCertText)
	}
	store := hostkey.KeyStore{Path: filepath.Join(o.Home, hostkey.KeyFile), Protector: o.Protector}
	private, err := store.Load()
	if err != nil {
		return nil, errors.New(HostKeyUnusableText)
	}
	if hostkey.EncodeKey(private.Public().(ed25519.PublicKey)) != public {
		return nil, errors.New(HostKeyMismatchText)
	}
	return &hostLicense{bundle: bundle, public: public, private: private}, nil
}

// Answer signs the proof of request for a VM, together with this PC's time, and hands over the host's
// certificate. The time is part of the signature, so a VM can tell how old the proof is.
func (h *HostResponder) Answer(request hostkey.Request) (hostkey.Answer, error) {
	license, err := h.Check()
	if err != nil {
		return hostkey.Answer{}, err
	}
	certificate, err := json.Marshal(struct {
		Payload   string `json:"payload"`
		Signature string `json:"signature"`
	}{license.bundle.Payload, license.bundle.Signature})
	if err != nil {
		return hostkey.Answer{}, err
	}
	at := int64(h.options.Clock())
	return hostkey.Answer{Certificate: certificate, Time: at, Signature: hostkey.Sign(license.private, request, at)}, nil
}
