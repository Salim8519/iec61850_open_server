package licensing

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Timing of the offline proof that this computer's security chip holds the license's
// attestation key (ADR-0098).
const (
	// ChipProofCacheSeconds is how long a successful proof is reused, so a status that is
	// read many times does not ask the chip every time.
	ChipProofCacheSeconds = 60
	// ChipGraceSeconds is how long a chip that proved the license a moment ago and is
	// momentarily busy or unreachable still counts; a computer that never proved it never does.
	ChipGraceSeconds = 600
)

// chipRetryDelay is the pause before the one retry of a failed proof (a busy chip).
var chipRetryDelay = 300 * time.Millisecond

// chipState remembers the last proof made by this process.
type chipState struct {
	mu       sync.Mutex // also keeps two proofs from asking the chip at once
	provedAt float64
	key      string // the "ak" of the certificate that was proven
}

// proveChip checks that this computer's chip signs a fresh nonce with the attestation key
// the certificate names (ak, base64 TPMT_PUBLIC): the chip loads the key from ak.blob and
// signs, then software verifies the signature with ak. A copy of the license on another
// computer has no such key.
func (s *Service) proveChip(ak string) error {
	now := s.options.Clock()
	s.chip.mu.Lock()
	defer s.chip.mu.Unlock()
	fresh := func(limit float64) bool {
		age := now - s.chip.provedAt
		return s.chip.provedAt > 0 && s.chip.key == ak && age >= 0 && age < limit
	}
	if fresh(ChipProofCacheSeconds) {
		return nil
	}
	err := s.proveOnce(ak)
	if err != nil {
		time.Sleep(chipRetryDelay)
		err = s.proveOnce(ak)
	}
	if err == nil {
		s.chip.provedAt, s.chip.key = now, ak
		return nil
	}
	if fresh(ChipGraceSeconds) {
		s.options.Logf("The security chip is busy (%v); the last proof still counts.", err)
		return nil
	}
	return err
}

func (s *Service) proveOnce(ak string) error {
	public, err := base64.StdEncoding.DecodeString(ak)
	if err != nil || len(public) == 0 {
		return errors.New("the license names an unreadable attestation key")
	}
	blob, err := os.ReadFile(s.akFile)
	if err != nil || len(blob) == 0 {
		return errors.New("this computer holds no attestation key for the license")
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	proof, err := s.options.Chip.Prove(blob, nonce)
	if err != nil {
		return fmt.Errorf("the chip did not sign: %w", err)
	}
	if err := s.options.Chip.Verify(public, nonce, proof); err != nil {
		return fmt.Errorf("the chip's answer does not match the license: %w", err)
	}
	return nil
}

// writeBinary replaces path with data byte for byte (no newline translation), through a
// temporary file so a reader never sees half of it.
func writeBinary(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o666); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
