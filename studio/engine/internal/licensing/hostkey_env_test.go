package licensing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/licensing/hostkey"
)

// The VM host key tests (ADR-0099) build their own signer, registry and homes, so they stand on
// nothing but the contract: a certificate is {payload, signature}, the payload is JSON with
// alphabetical keys and an optional "host_key".

type hkRegistry struct {
	mu     *sync.Mutex
	values map[string]string
}

func newHKRegistry() hkRegistry { return hkRegistry{mu: &sync.Mutex{}, values: map[string]string{}} }

func (r hkRegistry) Get(name string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.values[name]
	return value, ok
}

func (r hkRegistry) Set(name, text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[name] = text
}

// hkTempDir is t.TempDir() that keeps trying to remove the folder: on Windows a virus scanner can
// hold a fresh file for a moment ("The directory is not empty").
func hkTempDir(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("", "license-hostkey-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for attempt := 0; attempt < 50; attempt++ {
			if os.RemoveAll(root) == nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	})
	return root
}

// hkSigner signs certificates like the license server does.
type hkSigner struct {
	private ed25519.PrivateKey
	public  ed25519.PublicKey
}

func newHKSigner(t *testing.T) *hkSigner {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return &hkSigner{private: private, public: public}
}

func (s *hkSigner) keys() map[string][]byte { return map[string][]byte{KeyID(s.public): s.public} }

// sign makes a certificate: the defaults of a lifetime license, overridden by fields.
func (s *hkSigner) sign(t *testing.T, fields map[string]any) Bundle {
	t.Helper()
	payload := map[string]any{"email": "", "expires_at": 0, "hwid": strings.Repeat("a", 64), "issued_at": 1_700_000_000,
		"kid": KeyID(s.public), "license_id": "SUBS-HOST-HOST-HOST-HOST", "max_machines": 1, "type": "lifetime", "version": 1}
	for key, value := range fields {
		if value == nil {
			delete(payload, key)
		} else {
			payload[key] = value
		}
	}
	text, err := json.Marshal(payload) // encoding/json sorts map keys: the server's canonical order
	if err != nil {
		t.Fatal(err)
	}
	return Bundle{Payload: string(text), Signature: b64url(ed25519.Sign(s.private, text))}
}

func b64url(data []byte) string { return base64.RawURLEncoding.EncodeToString(data) }

// hkHost is a licensed PC: a license home with host_key.dpapi and license.cert.
type hkHost struct {
	t         *testing.T
	home      string
	hwid      string
	signer    *hkSigner
	protector hostkey.FakeProtector
	registry  hkRegistry
	now       *float64
	skew      float64 // seconds this PC's clock is ahead of the test clock
}

func newHKHost(t *testing.T, signer *hkSigner) *hkHost {
	t.Helper()
	moment := 1_760_000_000.0
	return &hkHost{t: t, home: hkTempDir(t), hwid: strings.Repeat("a", 64), signer: signer,
		protector: hostkey.FakeProtector{Owner: "alice@HOST"}, registry: newHKRegistry(), now: &moment}
}

func (h *hkHost) store() hostkey.KeyStore {
	return hostkey.KeyStore{Path: filepath.Join(h.home, hostkey.KeyFile), Protector: h.protector}
}

func (h *hkHost) publicKey() string {
	text, err := h.store().PublicText()
	if err != nil {
		h.t.Fatal(err)
	}
	return text
}

// install writes license.cert for this PC with this PC's host key, overridden by fields.
func (h *hkHost) install(fields map[string]any) Bundle {
	h.t.Helper()
	merged := map[string]any{"hwid": h.hwid, "host_key": h.publicKey()}
	for key, value := range fields {
		merged[key] = value
	}
	bundle := h.signer.sign(h.t, merged)
	data, _ := json.MarshalIndent(map[string]string{"payload": bundle.Payload, "signature": bundle.Signature}, "", "  ")
	if err := os.WriteFile(filepath.Join(h.home, "license.cert"), data, 0o666); err != nil {
		h.t.Fatal(err)
	}
	return bundle
}

func (h *hkHost) responder() *HostResponder {
	return NewHostResponder(HostOptions{Home: h.home, Keys: h.signer.keys(), HWID: func() (string, error) { return h.hwid, nil },
		Protector: h.protector, Registry: h.registry, Clock: func() float64 { return *h.now + h.skew }})
}
