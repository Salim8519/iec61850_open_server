package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"

	"golang.org/x/crypto/argon2"
)

// The sealed helper is this program's own bytes + an encrypted vault + a trailer:
// [vault JSON][8-byte little-endian vault length][16-byte magic]. Windows ignores
// data appended after the executable image, so the result is one portable file.
var vaultMagic = []byte("SCADABENCH-VAULT")

const minPasswordLength = 8

// Secrets is the only thing the vault holds: the owner signing key. It never touches
// the network; the license server's own key never leaves the server.
type Secrets struct {
	Seed    string `json:"seed"` // Ed25519 private seed, hex
	KeyID   string `json:"kid"`  // 8-hex id the Studio app uses to pick the public key
	Created string `json:"created"`
}

type kdfParams struct {
	Time    uint32 `json:"t"`
	Memory  uint32 `json:"m"` // KiB
	Threads uint8  `json:"p"`
}

type sealed struct {
	Version int       `json:"v"`
	KDF     kdfParams `json:"kdf"`
	Salt    []byte    `json:"salt"`
	Nonce   []byte    `json:"nonce"`
	Data    []byte    `json:"data"`
}

// Argon2id at 256 MiB x 4 passes: every password guess costs about a second and a
// quarter-gigabyte of memory, which makes offline guessing of a good password hopeless.
var defaultKDF = kdfParams{Time: 4, Memory: 256 * 1024, Threads: 4}

var errWrongPassword = errors.New("wrong password")

func deriveKey(password string, salt []byte, p kdfParams) []byte {
	return argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, 32)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// sealSecrets encrypts the secrets with a key derived from the password.
func sealSecrets(secrets Secrets, password string, p kdfParams) ([]byte, error) {
	if len(password) < minPasswordLength {
		return nil, errors.New("the password must be at least 8 characters")
	}
	plain, err := json.Marshal(secrets)
	if err != nil {
		return nil, err
	}
	box := sealed{Version: 1, KDF: p, Salt: make([]byte, 16)}
	if _, err := rand.Read(box.Salt); err != nil {
		return nil, err
	}
	aead, err := newGCM(deriveKey(password, box.Salt, p))
	if err != nil {
		return nil, err
	}
	box.Nonce = make([]byte, aead.NonceSize())
	if _, err := rand.Read(box.Nonce); err != nil {
		return nil, err
	}
	box.Data = aead.Seal(nil, box.Nonce, plain, vaultMagic)
	return json.Marshal(box)
}

// openSecrets decrypts a vault; a wrong password or any altered byte fails.
func openSecrets(vault []byte, password string) (Secrets, error) {
	var box sealed
	var secrets Secrets
	if err := json.Unmarshal(vault, &box); err != nil || box.Version != 1 {
		return secrets, errors.New("the helper's built-in vault is damaged")
	}
	aead, err := newGCM(deriveKey(password, box.Salt, box.KDF))
	if err != nil {
		return secrets, err
	}
	if len(box.Nonce) != aead.NonceSize() {
		return secrets, errors.New("the helper's built-in vault is damaged")
	}
	plain, err := aead.Open(nil, box.Nonce, box.Data, vaultMagic)
	if err != nil {
		return secrets, errWrongPassword
	}
	err = json.Unmarshal(plain, &secrets)
	return secrets, err
}

// appendVault returns program bytes with the vault and trailer attached.
func appendVault(program, vault []byte) []byte {
	out := bytes.Clone(stripVault(program))
	out = append(out, vault...)
	out = binary.LittleEndian.AppendUint64(out, uint64(len(vault)))
	return append(out, vaultMagic...)
}

// splitVault returns the attached vault, or nil when the program has none.
func splitVault(program []byte) []byte {
	trailer := len(vaultMagic) + 8
	if len(program) < trailer || !bytes.Equal(program[len(program)-len(vaultMagic):], vaultMagic) {
		return nil
	}
	size := binary.LittleEndian.Uint64(program[len(program)-trailer:])
	if size == 0 || size > uint64(len(program)-trailer) {
		return nil
	}
	end := len(program) - trailer
	return program[end-int(size) : end]
}

func stripVault(program []byte) []byte {
	vault := splitVault(program)
	if vault == nil {
		return program
	}
	return program[:len(program)-len(vaultMagic)-8-len(vault)]
}

func ownVault() ([]byte, []byte, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, nil, err
	}
	program, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	return program, splitVault(program), nil
}
