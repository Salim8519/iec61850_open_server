package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Payload mirrors license-server/internal/certs.Payload field for field. Fields are in
// alphabetical order, so encoding/json gives the same canonical bytes as the server.
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

// Bundle is the license.cert file, exactly as the server and the Studio app write it.
type Bundle struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

const ownerEmail = "owner@scadabench.com"

// clockSlack backdates issued_at so a target PC whose clock is a little behind this
// one still accepts the file (the app allows only 5 minutes of rollback).
const clockSlack = time.Hour

// Signer holds the decrypted owner key for this session only.
type Signer struct {
	private ed25519.PrivateKey
	keyID   string
}

func keyIDFor(public ed25519.PublicKey) string {
	digest := sha256.Sum256(public)
	return hex.EncodeToString(digest[:4])
}

func newOwnerKey() (Secrets, ed25519.PublicKey, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Secrets{}, nil, err
	}
	return Secrets{Seed: hex.EncodeToString(private.Seed()), KeyID: keyIDFor(public),
		Created: time.Now().UTC().Format(time.RFC3339)}, public, nil
}

func newSigner(secrets Secrets) (*Signer, error) {
	seed, err := hex.DecodeString(secrets.Seed)
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, errors.New("the helper's signing key is damaged")
	}
	private := ed25519.NewKeyFromSeed(seed)
	clear(seed)
	signer := &Signer{private: private, keyID: keyIDFor(private.Public().(ed25519.PublicKey))}
	if signer.keyID != secrets.KeyID {
		return nil, errors.New("the helper's signing key does not match its key id")
	}
	return signer, nil
}

// wipe overwrites the private key in memory when the session ends.
func (s *Signer) wipe() { clear(s.private) }

func (s *Signer) publicHex() string {
	return hex.EncodeToString(s.private.Public().(ed25519.PublicKey))
}

// licenseFor signs a certificate binding one machine. days == 0 means lifetime.
func (s *Signer) licenseFor(hwid string, days int, now time.Time) (Bundle, Payload, error) {
	if !validHWID(hwid) {
		return Bundle{}, Payload{}, errors.New("a Machine ID is 64 characters of 0-9 and a-f; copy it again")
	}
	if days < 0 || days > 999 {
		return Bundle{}, Payload{}, errors.New("days must be from 1 to 999")
	}
	id := make([]byte, 8)
	if _, err := rand.Read(id); err != nil {
		return Bundle{}, Payload{}, err
	}
	payload := Payload{Email: ownerEmail, HWID: hwid, IssuedAt: now.Add(-clockSlack).Unix(), KeyID: s.keyID,
		LicenseID: "OWNER-" + strings.ToUpper(hex.EncodeToString(id)), MaxMachines: 1, Type: "lifetime", Version: 1}
	if days > 0 {
		payload.Type, payload.ExpiresAt = "trial", now.Add(time.Duration(days)*24*time.Hour).Unix()
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return Bundle{}, Payload{}, err
	}
	bundle := Bundle{Payload: string(raw),
		Signature: base64.RawURLEncoding.EncodeToString(ed25519.Sign(s.private, raw))}
	if err := verifyBundle(s.private.Public().(ed25519.PublicKey), bundle); err != nil {
		return Bundle{}, Payload{}, fmt.Errorf("self-check failed, nothing was written: %w", err)
	}
	return bundle, payload, nil
}

// verifyBundle re-checks a freshly made certificate the way the Studio app will.
func verifyBundle(public ed25519.PublicKey, bundle Bundle) error {
	signature, err := base64.RawURLEncoding.DecodeString(bundle.Signature)
	if err != nil || !ed25519.Verify(public, []byte(bundle.Payload), signature) {
		return errors.New("signature does not verify")
	}
	return nil
}
