package hostkey

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
)

// KeyStore keeps this PC's host key pair: the Ed25519 seed encrypted with DPAPI in one file.
// The public key is derived from the seed, so the file is all there is to keep.
type KeyStore struct {
	Path      string // host_key.dpapi in the license home
	Protector Protector
}

// ErrNoKey is returned when the PC holds no host key at all.
var ErrNoKey = errors.New("this computer holds no virtual machine key")

// Load decrypts and returns the private key. A file copied from another PC or user (or damaged)
// fails with ErrUnprotect, a missing file with ErrNoKey.
func (k KeyStore) Load() (ed25519.PrivateKey, error) {
	if k.Protector == nil {
		return nil, errors.New("this computer cannot protect a virtual machine key")
	}
	blob, err := os.ReadFile(k.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNoKey
	}
	if err != nil {
		return nil, err
	}
	seed, err := k.Protector.Unprotect(blob)
	if err != nil {
		return nil, err
	}
	if len(seed) != ed25519.SeedSize {
		return nil, ErrUnprotect
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

// LoadOrCreate returns the key, making a new pair when there is none or the old one cannot be
// opened here (a copy from another PC, or a Windows user that was recreated). It is only used when
// a license is activated, which hands the new public key to the server.
func (k KeyStore) LoadOrCreate() (ed25519.PrivateKey, error) {
	if private, err := k.Load(); err == nil {
		return private, nil
	}
	if k.Protector == nil {
		return nil, errors.New("this computer cannot protect a virtual machine key")
	}
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	blob, err := k.Protector.Protect(private.Seed())
	if err != nil {
		return nil, err
	}
	if err := writeAtomic(k.Path, blob); err != nil {
		return nil, err
	}
	return private, nil
}

// PublicText is the base64url text of the public key, creating the pair when needed.
func (k KeyStore) PublicText() (string, error) {
	private, err := k.LoadOrCreate()
	if err != nil {
		return "", err
	}
	return EncodeKey(private.Public().(ed25519.PublicKey)), nil
}

// writeAtomic replaces path byte for byte through a temporary file, so a reader never sees half.
func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
