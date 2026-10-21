package hostkey

import (
	"bytes"
	"errors"
)

// Protector encrypts a secret so only this Windows user on this PC can read it back (DPAPI).
type Protector interface {
	Protect(plain []byte) ([]byte, error)
	Unprotect(blob []byte) ([]byte, error)
}

// ErrUnprotect is returned when a blob cannot be decrypted here: another PC or user, damage.
var ErrUnprotect = errors.New("this PC cannot decrypt the virtual machine key")

// FakeProtector is the test double of DPAPI: a blob opens only for the owner that made it,
// exactly like a blob copied to another PC or user cannot be decrypted. Owner is any text.
type FakeProtector struct{ Owner string }

const fakeMagic = "FAKE-DPAPI|"

// Protect wraps plain for the owner (no real encryption: tests only).
func (f FakeProtector) Protect(plain []byte) ([]byte, error) {
	return append([]byte(fakeMagic+f.Owner+"|"), plain...), nil
}

// Unprotect opens a blob made for the same owner and refuses every other one.
func (f FakeProtector) Unprotect(blob []byte) ([]byte, error) {
	head := []byte(fakeMagic + f.Owner + "|")
	if !bytes.HasPrefix(blob, head) {
		return nil, ErrUnprotect
	}
	return append([]byte(nil), blob[len(head):]...), nil
}
