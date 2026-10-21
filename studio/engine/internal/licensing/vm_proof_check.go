package licensing

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"studio-engine/internal/licensing/hostkey"
	"studio-engine/internal/licensing/pytext"
)

// Texts of the VM proof's clocks (docs/specs/licensing.md LIC-81). Plain words, no jargon.
const (
	VMClockSkewText = "The clock of this virtual machine and the clock of its host PC differ by more than 10 minutes. " +
		"Set the correct time on both and try again."
	VMProofOldText = "The confirmation of the host PC is out of date. Start ScadaBench on the host PC and try again."
)

// verifyProof is every rule of the contract for a host's proof, whether it was just received over
// the network (fresh) or is read back from vm_proof.json: the certificate verifies with the release
// keys, is a lifetime license that is not cancelled here and names a host key, and the host key's
// signature verifies for this request AND the exact time the host signed. A fresh answer must also
// carry a time within ten minutes of this computer's clock (a host whose clock is wrong would hand
// out proofs that are valid for the wrong period). Both end with proofLive: not from the future, not
// older than 24 hours, and not older than 24 hours before the newest time this computer has seen.
func (s *Service) verifyProof(certificate json.RawMessage, request hostkey.Request, signature string, at int64,
	fresh bool) (*vmProof, error) {
	bundle, err := ParseBundle(string(certificate))
	if err != nil {
		return nil, err
	}
	payload, err := VerifyBundle(bundle, s.options.Keys)
	if err != nil {
		return nil, err
	}
	version, damaged, err := payloadInt(payload, "version", true)
	if err != nil || damaged || version != 1 {
		return nil, errors.New(Unsupported)
	}
	if plan, _ := payload.Get("type"); plan != any("lifetime") {
		return nil, errors.New(HostNotLifetimeText)
	}
	now := s.now()
	issued, damaged, err := payloadInt(payload, "issued_at", false)
	if err != nil || damaged {
		return nil, errors.New(Damaged)
	}
	expires, _, _ := payloadInt(payload, "expires_at", true)
	switch {
	case now+ClockToleranceSeconds < max(s.anchor.MaxSeen(), issued):
		return nil, errors.New(ClockRollback)
	case expires > 0 && now >= expires:
		return nil, errors.New(LicenseExpired)
	}
	licenseID, _ := payload.Get("license_id")
	if s.revoked.Contains(licenseID) {
		return nil, errors.New(RevokedText)
	}
	hostText := certHostKey(payload)
	public, err := hostkey.DecodeKey(hostText)
	if err != nil {
		return nil, errors.New("the host's license has no virtual machine key")
	}
	if fresh && (at < now-hostkey.ProofSkewSeconds || at > now+hostkey.ProofSkewSeconds) {
		return nil, errors.New(VMClockSkewText)
	}
	if !hostkey.Verify(public, request, at, signature) {
		return nil, errors.New("the host's signature does not match this request")
	}
	if err := s.proofLive(now, at); err != nil {
		return nil, err
	}
	return &vmProof{at: at, licenseID: licenseID, hostKey: hostText}, nil
}

// proofLive says whether a proof the host signed at time at may license this computer now: at most
// ten minutes ahead of this clock, at most 24 hours old by this clock, and at most 24 hours older than
// the newest time this computer has ever seen (the license's own clock-rollback anchor), so setting
// the clock back cannot bring an old proof back to life.
func (s *Service) proofLive(now, at int64) error {
	if at > now+hostkey.ProofSkewSeconds || now-at > hostkey.ProofMaxAgeSeconds {
		return errors.New(VMProofOldText)
	}
	if s.anchor.MaxSeen()-at > hostkey.ProofMaxAgeSeconds {
		return errors.New(ClockRollback)
	}
	return nil
}

// readVMProof reads vm_proof.json back and applies every rule again (the file can be edited by
// anyone): the machine ID in it must be this computer's, and the time in it is the one the host
// signed, so changing it breaks the signature. A missing or damaged file is simply no proof.
func (s *Service) readVMProof() *vmProof {
	text, err := pytext.ReadText(os.ReadFile(filepath.Join(s.options.Home, VMProofFile)))
	if err != nil {
		return nil
	}
	var document vmProofDocument
	if json.Unmarshal([]byte(text), &document) != nil || document.Time <= 0 {
		return nil
	}
	request := hostkey.Request{Nonce: document.Nonce, VMHWID: document.VMHWID}
	hwid, err := s.HWID()
	if err != nil || request.Validate() != nil || request.VMHWID != hwid {
		return nil
	}
	proof, err := s.verifyProof(document.Certificate, request, document.Signature, document.Time, false)
	if err != nil {
		return nil
	}
	return proof
}

// storedVMProof is the proof this process holds, read from vm_proof.json once if it holds none yet.
func (s *Service) storedVMProof() *vmProof {
	s.vm.mu.Lock()
	defer s.vm.mu.Unlock()
	if s.vm.proof == nil && !s.vm.loaded {
		s.vm.loaded = true
		s.vm.proof = s.readVMProof()
	}
	return s.vm.proof
}
