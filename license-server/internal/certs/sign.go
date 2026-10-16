// Package certs signs license certificates with the server's Ed25519 key.
package certs

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Payload is the signed license content. Field order is alphabetical, so the
// encoding/json output is the canonical byte string every client verifies.
type Payload struct {
	Email       string `json:"email"`
	ExpiresAt   int64  `json:"expires_at"`
	HWID        string `json:"hwid"`
	IssuedAt    int64  `json:"issued_at"`
	KeyID       string `json:"kid"`
	LicenseID   string `json:"license_id"`
	MaxMachines int    `json:"max_machines"`
	Type        string `json:"type"`
	Version     int    `json:"version"`
}

// Bundle is the license.cert file: the exact signed string plus its signature.
type Bundle struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

// Signer owns the private key and its short key identifier.
type Signer struct {
	private ed25519.PrivateKey
	KeyID   string
}

// KeyPath is the private seed file inside the data directory.
func KeyPath(dataDir string) string { return filepath.Join(dataDir, "signing.key") }

// KeyIDFor derives the 8-hex key identifier clients use to pick a public key.
func KeyIDFor(public ed25519.PublicKey) string {
	digest := sha256.Sum256(public)
	return hex.EncodeToString(digest[:4])
}

// Generate creates a new seed file; it refuses to overwrite unless force is set.
func Generate(dataDir string, force bool) (*Signer, error) {
	path := KeyPath(dataDir)
	if _, err := os.Stat(path); err == nil && !force {
		return nil, errors.New("signing.key already exists; back it up and pass --force to replace it")
	}
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	seed := hex.EncodeToString(private.Seed())
	if err := os.WriteFile(path, []byte(seed+"\n"), 0o600); err != nil {
		return nil, err
	}
	return fromPrivate(private), nil
}

// Load reads the seed file.
func Load(dataDir string) (*Signer, error) {
	raw, err := os.ReadFile(KeyPath(dataDir))
	if err != nil {
		return nil, errors.New("no signing key yet; run `license-server configure` or `license-server keygen`")
	}
	seed, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, errors.New("signing.key is damaged; restore it from your backup")
	}
	return fromPrivate(ed25519.NewKeyFromSeed(seed)), nil
}

func fromPrivate(private ed25519.PrivateKey) *Signer {
	public := private.Public().(ed25519.PublicKey)
	return &Signer{private: private, KeyID: KeyIDFor(public)}
}

// PublicHex returns the 64-hex public key the Windows release embeds.
func (s *Signer) PublicHex() string {
	return hex.EncodeToString(s.private.Public().(ed25519.PublicKey))
}

// Sign stamps the key id, serialises the payload and signs the exact bytes.
func (s *Signer) Sign(p Payload) (Bundle, error) {
	p.KeyID = s.KeyID
	p.Version = 1
	raw, err := json.Marshal(p)
	if err != nil {
		return Bundle{}, err
	}
	signature := ed25519.Sign(s.private, raw)
	return Bundle{Payload: string(raw), Signature: base64.RawURLEncoding.EncodeToString(signature)}, nil
}

// Verify checks a bundle against a public key (used by tests and the admin CLI).
func Verify(public ed25519.PublicKey, b Bundle) (Payload, error) {
	signature, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(b.Signature, "="))
	if err != nil || !ed25519.Verify(public, []byte(b.Payload), signature) {
		return Payload{}, errors.New("signature does not match")
	}
	var p Payload
	err = json.Unmarshal([]byte(b.Payload), &p)
	return p, err
}

// Public returns the signer's public key.
func (s *Signer) Public() ed25519.PublicKey { return s.private.Public().(ed25519.PublicKey) }

// keyAlphabet avoids look-alike characters (0/O, 1/I/L) so keys survive dictation.
const keyAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// NewLicenseKey returns SUBS-XXXX-XXXX-XXXX-XXXX with 80 bits of randomness.
func NewLicenseKey() string {
	groups := make([]string, 4)
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic("system random source failed: " + err.Error())
	}
	for g := range groups {
		chunk := make([]byte, 4)
		for i := range chunk {
			chunk[i] = keyAlphabet[int(buf[g*4+i])%len(keyAlphabet)]
		}
		groups[g] = string(chunk)
	}
	return "SUBS-" + strings.Join(groups, "-")
}

// NormalizeKey upper-cases and trims user-typed keys.
func NormalizeKey(key string) string { return strings.ToUpper(strings.TrimSpace(key)) }

// ValidHWID accepts exactly 64 lowercase hex characters.
func ValidHWID(hwid string) bool {
	if len(hwid) != 64 {
		return false
	}
	_, err := hex.DecodeString(hwid)
	return err == nil && strings.ToLower(hwid) == hwid
}
